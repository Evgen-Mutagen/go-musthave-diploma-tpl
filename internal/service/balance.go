package service

import (
	"context"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/model"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/repository"
)

type BalanceService interface {
	GetBalance(ctx context.Context, userID int64) (*model.UserBalance, error)
}

type balanceService struct {
	userRepo     repository.UserRepository
	orderRepo    repository.OrderRepository
	withdrawRepo repository.WithdrawalRepository
}

func NewBalanceService(
	userRepo repository.UserRepository,
	orderRepo repository.OrderRepository,
	withdrawRepo repository.WithdrawalRepository,
) BalanceService {
	return &balanceService{
		userRepo:     userRepo,
		orderRepo:    orderRepo,
		withdrawRepo: withdrawRepo,
	}
}

func (s *balanceService) GetBalance(ctx context.Context, userID int64) (*model.UserBalance, error) {
	return s.userRepo.GetBalance(ctx, userID)
}
