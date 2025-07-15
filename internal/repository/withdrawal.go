package repository

import (
	"context"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/model"
)

type WithdrawalRepository interface {
	Create(ctx context.Context, withdrawal *model.Withdrawal) error
	GetByUserID(ctx context.Context, userID int64) ([]*model.Withdrawal, error)
}

type withdrawalRepository struct {
	db *Database
}

func NewWithdrawalRepository(db *Database) WithdrawalRepository {
	return &withdrawalRepository{db: db}
}

func (r *withdrawalRepository) Create(ctx context.Context, withdrawal *model.Withdrawal) error {
	query := `INSERT INTO withdrawals (order_number, user_id, sum, processed_at) 
              VALUES ($1, $2, $3, $4)`
	_, err := r.db.db.ExecContext(ctx, query, withdrawal.Order, withdrawal.UserID, withdrawal.Sum, withdrawal.ProcessedAt)
	return err
}

func (r *withdrawalRepository) GetByUserID(ctx context.Context, userID int64) ([]*model.Withdrawal, error) {
	query := `SELECT order_number, sum, processed_at 
              FROM withdrawals 
              WHERE user_id = $1
              ORDER BY processed_at DESC`
	rows, err := r.db.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []*model.Withdrawal
	for rows.Next() {
		var w model.Withdrawal
		if err := rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, &w)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return withdrawals, nil
}
