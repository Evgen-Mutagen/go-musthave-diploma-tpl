package controller

import (
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/service"
	"go.uber.org/zap"
	"io"
	"net/http"

	"github.com/go-chi/render"
)

type OrderController struct {
	orderService service.OrderService
	logger       *zap.Logger
}

func NewOrderController(orderService service.OrderService, logger *zap.Logger) *OrderController {
	return &OrderController{
		orderService: orderService,
		logger:       logger,
	}
}

func (c *OrderController) UploadOrder(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int64)

	body, err := io.ReadAll(r.Body)
	if err != nil {
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
		switch err {
		case service.ErrOrderAlreadyUploaded:
			w.WriteHeader(http.StatusOK)
		case service.ErrOrderUploadedByOtherUser:
			http.Error(w, "Order already uploaded by another user", http.StatusConflict)
		case service.ErrInvalidOrderNumber:
			http.Error(w, "Invalid order number", http.StatusUnprocessableEntity)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (c *OrderController) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
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
