package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/model"
)

type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	GetByNumber(ctx context.Context, number string) (*model.Order, error)
	GetByUserID(ctx context.Context, userID int64) ([]*model.Order, error)
	Update(ctx context.Context, order *model.Order) error
	GetUnprocessedOrders(ctx context.Context) ([]*model.Order, error)
}

type orderRepository struct {
	db *Database
}

func NewOrderRepository(db *Database) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, order *model.Order) error {
	query := `INSERT INTO orders (number, user_id, status, uploaded_at)
              VALUES ($1, $2, $3, $4)`

	_, err := r.db.db.ExecContext(ctx, query,
		order.Number,
		order.UserID,
		order.Status,
		order.UploadedAt,
	)
	return err
}

func (r *orderRepository) GetByNumber(ctx context.Context, number string) (*model.Order, error) {
	order := &model.Order{}
	query := `SELECT number, user_id, status, accrual, uploaded_at 
              FROM orders WHERE number = $1`

	err := r.db.db.QueryRowContext(ctx, query, number).Scan(
		&order.Number,
		&order.UserID,
		&order.Status,
		&order.Accrual,
		&order.UploadedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

func (r *orderRepository) GetByUserID(ctx context.Context, userID int64) ([]*model.Order, error) {
	if r.db == nil || r.db.db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	query := `SELECT number, status, accrual, uploaded_at 
              FROM orders 
              WHERE user_id = $1
              ORDER BY uploaded_at DESC`

	rows, err := r.db.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var order model.Order
		var accrual sql.NullFloat64

		if err := rows.Scan(
			&order.Number,
			&order.Status,
			&accrual,
			&order.UploadedAt,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if accrual.Valid {
			order.Accrual = accrual.Float64
		}

		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return orders, nil
}

func (r *orderRepository) Update(ctx context.Context, order *model.Order) error {
	query := `UPDATE orders SET status = $1, accrual = $2 WHERE number = $3`
	_, err := r.db.db.ExecContext(ctx, query, order.Status, order.Accrual, order.Number)
	return err
}

func (r *orderRepository) GetUnprocessedOrders(ctx context.Context) ([]*model.Order, error) {
	query := `SELECT number, user_id, status, accrual, uploaded_at 
              FROM orders 
              WHERE status IN ('NEW', 'PROCESSING')
              ORDER BY uploaded_at ASC`
	rows, err := r.db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var order model.Order
		if err := rows.Scan(
			&order.Number,
			&order.UserID,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}
