package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByLogin(ctx context.Context, login string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
	UpdateBalance(ctx context.Context, userID int64, amount float64) error
	GetBalance(ctx context.Context, userID int64) (*model.UserBalance, error)
	BeginTx(ctx context.Context) (*sql.Tx, error)
}

type userRepository struct {
	db *Database
}

func NewUserRepository(db *Database) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id, created_at`
	err := r.db.db.QueryRowContext(ctx, query, user.Login, user.PasswordHash).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *userRepository) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, login, password_hash, created_at FROM users WHERE login = $1`
	err := r.db.db.QueryRowContext(ctx, query, login).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, login, password_hash, created_at FROM users WHERE id = $1`
	err := r.db.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *userRepository) UpdateBalance(ctx context.Context, userID int64, amount float64) error {
	query := `UPDATE users 
              SET balance = balance + $1, 
                  withdrawn = withdrawn + CASE WHEN $1 < 0 THEN -$1 ELSE 0 END
              WHERE id = $2`
	_, err := r.db.db.ExecContext(ctx, query, amount, userID)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}
	return nil
}

func (r *userRepository) GetBalance(ctx context.Context, userID int64) (*model.UserBalance, error) {
	balance := &model.UserBalance{}
	query := `SELECT balance, withdrawn FROM users WHERE id = $1`
	err := r.db.db.QueryRowContext(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (r *userRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx)
}
