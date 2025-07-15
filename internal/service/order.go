package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/model"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/repository"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/util/luhn"
	"io"
	"net/http"
	"time"
)

var (
	ErrOrderAlreadyUploaded     = errors.New("order already uploaded by this user")
	ErrOrderUploadedByOtherUser = errors.New("order already uploaded by another user")
	ErrInvalidOrderNumber       = errors.New("invalid order number")
)

type OrderService interface {
	UploadOrder(ctx context.Context, userID int64, orderNumber string) error
	GetOrders(ctx context.Context, userID int64) ([]*model.Order, error)
	ProcessOrders(ctx context.Context) error
}

type orderService struct {
	orderRepo          repository.OrderRepository
	accrualSystemAddr  string
	processingInterval time.Duration
}

func NewOrderService(orderRepo repository.OrderRepository, accrualSystemAddr string) OrderService {
	return &orderService{
		orderRepo:          orderRepo,
		accrualSystemAddr:  accrualSystemAddr,
		processingInterval: 5 * time.Second,
	}
}

func (s *orderService) UploadOrder(ctx context.Context, userID int64, orderNumber string) error {
	if !luhn.Validate(orderNumber) {
		return ErrInvalidOrderNumber
	}

	existingOrder, err := s.orderRepo.GetByNumber(ctx, orderNumber)
	if err != nil {
		return err
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

	return s.orderRepo.Create(ctx, order)
}

func (s *orderService) GetOrders(ctx context.Context, userID int64) ([]*model.Order, error) {
	return s.orderRepo.GetByUserID(ctx, userID)
}

func (s *orderService) ProcessOrders(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.processingInterval):
			orders, err := s.orderRepo.GetUnprocessedOrders(ctx)
			if err != nil {
				continue
			}

			for _, order := range orders {
				if err := s.processOrder(ctx, order); err != nil {
					continue
				}
			}
		}
	}
}

func (s *orderService) processOrder(ctx context.Context, order *model.Order) error {
	url := fmt.Sprintf("%s/api/orders/%s", s.accrualSystemAddr, order.Number)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			duration, err := time.ParseDuration(retryAfter + "s")
			if err == nil {
				time.Sleep(duration)
			}
		}
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Order   string  `json:"order"`
		Status  string  `json:"status"`
		Accrual float64 `json:"accrual,omitempty"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	order.Status = result.Status
	if result.Accrual > 0 {
		order.Accrual = result.Accrual
	}

	return s.orderRepo.Update(ctx, order)
}
