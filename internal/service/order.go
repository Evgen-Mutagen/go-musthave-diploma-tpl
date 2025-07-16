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
	userRepo           repository.UserRepository
	accrualSystemAddr  string
	processingInterval time.Duration
	logger             *zap.Logger
}

func NewOrderService(
	repo repository.OrderRepository,
	accrualAddr string,
	userRepo repository.UserRepository,
	logger *zap.Logger,
) core.OrderService {
	return &orderService{
		orderRepo:          repo,
		userRepo:           userRepo,
		accrualSystemAddr:  accrualAddr,
		processingInterval: 5 * time.Second,
		logger:             logger,
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
	orders, err := s.orderRepo.GetUnprocessedOrders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unprocessed orders: %w", err)
	}

	for _, order := range orders {
		if order.Status == "NEW" {
			order.Status = "PROCESSING"
			if err := s.orderRepo.Update(ctx, order); err != nil {
				s.logger.Error("Failed to update order status",
					zap.String("order", order.Number),
					zap.Error(err))
				continue
			}
		}

		status, accrual, err := s.getOrderStatusFromAccrual(ctx, order.Number)
		if err != nil {
			s.logger.Warn("Failed to get order status from accrual",
				zap.String("order", order.Number),
				zap.Error(err))
			continue
		}

		if status != order.Status || (status == "PROCESSED" && accrual != order.Accrual) {
			order.Status = status
			if status == "PROCESSED" {
				order.Accrual = accrual
				if err := s.userRepo.UpdateBalance(ctx, order.UserID, accrual); err != nil {
					s.logger.Error("Failed to update user balance",
						zap.Int64("user_id", order.UserID),
						zap.String("order", order.Number),
						zap.Error(err))
					continue
				}
			}

			if err := s.orderRepo.Update(ctx, order); err != nil {
				s.logger.Error("Failed to update order",
					zap.String("order", order.Number),
					zap.Error(err))
			}
		}
	}
	return nil
}

func (s *orderService) getOrderStatusFromAccrual(ctx context.Context, orderNumber string) (string, float64, error) {
	url := fmt.Sprintf("%s/api/orders/%s", s.accrualSystemAddr, orderNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result struct {
			Order   string  `json:"order"`
			Status  string  `json:"status"`
			Accrual float64 `json:"accrual,omitempty"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", 0, err
		}
		return result.Status, result.Accrual, nil

	case http.StatusNoContent:
		return "PROCESSING", 0, nil

	case http.StatusTooManyRequests:
		time.Sleep(time.Second * 5)
		return "PROCESSING", 0, nil

	default:
		return "INVALID", 0, nil
	}
}

func (s *orderService) processOrder(ctx context.Context, order *model.Order) error {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
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
			if err := s.userRepo.UpdateBalance(ctx, order.UserID, result.Accrual); err != nil {
				return err
			}
		}

		return s.orderRepo.Update(ctx, order)
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
