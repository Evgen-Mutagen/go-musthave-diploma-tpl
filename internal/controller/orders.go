package controller

import (
	"errors"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/core"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/middlewareinternal"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/service"
	"go.uber.org/zap"
	"io"
	"net/http"

	"github.com/go-chi/render"
)

type OrderController struct {
	orderService core.OrderProcessor
	logger       *zap.Logger
}

func NewOrderController(orderService core.OrderProcessor, logger *zap.Logger) *OrderController {
	return &OrderController{
		orderService: orderService,
		logger:       logger,
	}
}

func (c *OrderController) UploadOrder(w http.ResponseWriter, r *http.Request) {
	if c.logger == nil {
		panic("logger is not initialized")
	}

	userID, err := middlewareinternal.GetUserIDFromContext(r.Context())
	if err != nil {
		c.logger.Error("Failed to get user ID", zap.Error(err))
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
	userID, err := middlewareinternal.GetUserIDFromContext(r.Context())
	if err != nil {
		c.logger.Error("Failed to get user ID", zap.Error(err))
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
