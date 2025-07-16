package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/core"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/model"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/repository"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/util/luhn"
	"go.uber.org/zap"
	"net/http"
	"time"
)

var (
	ErrOrderAlreadyUploaded     = errors.New("order already uploaded by this user")
	ErrOrderUploadedByOtherUser = errors.New("order already uploaded by another user")
	ErrInvalidOrderNumber       = errors.New("invalid order number")
)

type orderService struct {
	orderRepo          repository.OrderRepository
	accrualSystemAddr  string
	processingInterval time.Duration
	logger             zap.Logger
	userRepo           repository.UserRepository
}

func NewOrderService(repo repository.OrderRepository, accrualAddr string) core.OrderService {
	return &orderService{
		orderRepo:          repo,
		accrualSystemAddr:  accrualAddr,
		processingInterval: 5 * time.Second,
	}
}

func (s *orderService) UploadOrder(ctx context.Context, userID int64, orderNumber string) error {
	if !luhn.Validate(orderNumber) {
		return ErrInvalidOrderNumber
	}

	existingOrder, err := s.orderRepo.GetByNumber(ctx, orderNumber)
	if err != nil {
		return fmt.Errorf("failed to check order: %w", err)
	}

	if existingOrder != nil {
		if existingOrder.UserID == userID {
			return ErrOrderAlreadyUploaded
		}
		return ErrOrderUploadedByOtherUser
	}

	order := &model.Order{
		Number:     orderNumber,
		UserID:     userID,
		Status:     "NEW",
		UploadedAt: time.Now(),
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

func (s *orderService) GetOrders(ctx context.Context, userID int64) ([]*model.Order, error) {
	return s.orderRepo.GetByUserID(ctx, userID)
}

func (s *orderService) ProcessOrders(ctx context.Context) error {
	ticker := time.NewTicker(s.processingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			orders, err := s.orderRepo.GetUnprocessedOrders(ctx)
			if err != nil {
				continue
			}

			for _, order := range orders {
				if err := s.processOrder(ctx, order); err != nil {
					s.logger.Error("Failed to process order",
						zap.String("order", order.Number),
						zap.Error(err))
					continue
				}
			}
		}
	}
}

func (s *orderService) processOrder(ctx context.Context, order *model.Order) error {
	// Добавляем таймаут для запроса к accrual
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/orders/%s", s.accrualSystemAddr, order.Number)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Обработка всех возможных статусов ответа
	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusTooManyRequests:
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			duration, _ := time.ParseDuration(retryAfter + "s")
			time.Sleep(duration)
		}
		return nil
	case http.StatusOK:
		var result struct {
			Order   string  `json:"order"`
			Status  string  `json:"status"`
			Accrual float64 `json:"accrual,omitempty"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}

		order.Status = result.Status
		if result.Accrual > 0 {
			order.Accrual = result.Accrual
			// Обновляем баланс пользователя
			if err := s.userRepo.UpdateBalance(ctx, order.UserID, result.Accrual); err != nil {
				return err
			}
		}

		return s.orderRepo.Update(ctx, order)
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
