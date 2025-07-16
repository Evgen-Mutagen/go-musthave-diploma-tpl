package core

import (
	"context"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/model"
)

type (
	AuthService interface {
		Register(ctx context.Context, login, password string) (*model.User, string, error)
		Login(ctx context.Context, login, password string) (*model.User, string, error)
		ValidateToken(tokenString string) (int64, error)
	}

	OrderService interface {
		UploadOrder(ctx context.Context, userID int64, orderNumber string) error
		GetOrders(ctx context.Context, userID int64) ([]*model.Order, error)
		ProcessOrders(ctx context.Context) error
	}
)
