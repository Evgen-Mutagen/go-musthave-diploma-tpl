package service

import (
	"context"
	"errors"
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

	balance, err := s.userRepo.GetBalance(ctx, userID)
	if err != nil {
		return err
	}

	if balance.Current < sum {
		return ErrWithdrawalInsufficientFunds
	}

	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"UPDATE users SET balance = balance - $1, withdrawn = withdrawn + $2 WHERE id = $3",
		sum, sum, userID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO withdrawals (order_number, user_id, sum, processed_at) VALUES ($1, $2, $3, $4)",
		orderNumber, userID, sum, time.Now()); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *withdrawalService) GetWithdrawals(ctx context.Context, userID int64) ([]*model.Withdrawal, error) {
	return s.withdrawalRepo.GetByUserID(ctx, userID)
}
