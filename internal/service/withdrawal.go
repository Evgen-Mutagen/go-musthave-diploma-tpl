package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/model"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/repository"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/util/luhn"
	"time"
)

var (
	ErrWithdrawalInsufficientFunds  = errors.New("insufficient funds")
	ErrWithdrawalInvalidOrderNumber = errors.New("invalid order number")
)

type WithdrawalService interface {
	Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error
	GetWithdrawals(ctx context.Context, userID int64) ([]*model.Withdrawal, error)
}

type withdrawalService struct {
	withdrawalRepo repository.WithdrawalRepository
	userRepo       repository.UserRepository
}

func NewWithdrawalService(
	withdrawalRepo repository.WithdrawalRepository,
	userRepo repository.UserRepository,
) WithdrawalService {
	return &withdrawalService{
		withdrawalRepo: withdrawalRepo,
		userRepo:       userRepo,
	}
}

func (s *withdrawalService) Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error {
	if !luhn.Validate(orderNumber) {
		return ErrWithdrawalInvalidOrderNumber
	}

	// Проверяем баланс в транзакции
	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	//balance, err := s.userRepo.GetBalance(ctx, userID)
	//if err != nil {
	//	return fmt.Errorf("failed to get balance: %w", err)
	//}

	//if balance.Current < sum {
	//	return ErrWithdrawalInsufficientFunds
	//}

	// Создаем запись о выводе
	withdrawal := &model.Withdrawal{
		Order:       orderNumber,
		UserID:      userID,
		Sum:         sum,
		ProcessedAt: time.Now(),
	}

	if err := s.withdrawalRepo.Create(ctx, withdrawal); err != nil {
		return fmt.Errorf("failed to create withdrawal: %w", err)
	}

	// Обновляем баланс
	if err := s.userRepo.UpdateBalance(ctx, userID, -sum); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	return tx.Commit()
}

func (s *withdrawalService) GetWithdrawals(ctx context.Context, userID int64) ([]*model.Withdrawal, error) {
	return s.withdrawalRepo.GetByUserID(ctx, userID)
}
