package controller

import (
	"errors"
	"fmt"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/core"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/service"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/types"
	"go.uber.org/zap"
	"io"
	"net/http"

	"github.com/go-chi/render"
)

type OrderController struct {
	orderService core.OrderService
	logger       *zap.Logger
}

func NewOrderController(orderService core.OrderService, logger *zap.Logger) *OrderController {
	return &OrderController{
		orderService: orderService,
		logger:       logger,
	}
}

func (c *OrderController) UploadOrder(w http.ResponseWriter, r *http.Request) {
	if c.logger == nil {
		fmt.Println("===== Логгер в OrderController равен nil! =====")
	} else {
		c.logger.Debug("===== UploadOrder called =====")
	}

	c.logger.Info("UploadOrder called")

	ctx := r.Context()
	c.logger.Debug("Context dump",
		zap.Any("types.UserIDKey", ctx.Value(types.UserIDKey)),
		zap.Any("string 'user_id'", ctx.Value("user_id")),
	)

	userID, ok := ctx.Value(types.UserIDKey).(int64)
	if !ok {
		c.logger.Error("UserID not found in context",
			zap.Any("types.UserIDKey", ctx.Value(types.UserIDKey)),
			zap.Any("string 'user_id'", ctx.Value("user_id")),
		)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	c.logger.Info("User authenticated", zap.Int64("user_id", userID))

	body, err := io.ReadAll(r.Body)
	if err != nil {
		c.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	orderNumber := string(body)
	if orderNumber == "" {
		http.Error(w, "Empty order number", http.StatusBadRequest)
		return
	}

	err = c.orderService.UploadOrder(r.Context(), userID, orderNumber)
	if err != nil {
		c.logger.Debug("Order upload error",
			zap.String("order", orderNumber),
			zap.Error(err))

		switch {
		case errors.Is(err, service.ErrOrderAlreadyUploaded):
			w.WriteHeader(http.StatusOK)
			return
		case errors.Is(err, service.ErrOrderUploadedByOtherUser):
			http.Error(w, "Order already uploaded by another user", http.StatusConflict)
			return
		case errors.Is(err, service.ErrInvalidOrderNumber):
			http.Error(w, "Invalid order number", http.StatusUnprocessableEntity)
			return
		default:
			c.logger.Error("Unexpected error in order upload",
				zap.String("order", orderNumber),
				zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func (c *OrderController) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(types.UserIDKey).(int64)
	if !ok {
		c.logger.Error("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := c.orderService.GetOrders(r.Context(), userID)
	if err != nil {
		c.logger.Error("Failed to get orders",
			zap.Int64("user_id", userID),
			zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	render.JSON(w, r, orders)
}
