package controller

import (
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/service"
	"net/http"

	"github.com/go-chi/render"
)

type WithdrawalController struct {
	withdrawalService service.WithdrawalService
}

func NewWithdrawalController(withdrawalService service.WithdrawalService) *WithdrawalController {
	return &WithdrawalController{withdrawalService: withdrawalService}
}

func (c *WithdrawalController) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int64)

	var request struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
	}

	if err := render.DecodeJSON(r.Body, &request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	err := c.withdrawalService.Withdraw(r.Context(), userID, request.Order, request.Sum)
	if err != nil {
		switch err {
		case service.ErrWithdrawalInsufficientFunds:
			http.Error(w, "Insufficient funds", http.StatusPaymentRequired)
		case service.ErrInvalidOrderNumber:
			http.Error(w, "Invalid order number", http.StatusUnprocessableEntity)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (c *WithdrawalController) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int64)

	withdrawals, err := c.withdrawalService.GetWithdrawals(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	render.JSON(w, r, withdrawals)
}
