package controller

import (
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/service"
	"net/http"

	"github.com/go-chi/render"
)

type BalanceController struct {
	balanceService service.BalanceService
}

func NewBalanceController(balanceService service.BalanceService) *BalanceController {
	return &BalanceController{balanceService: balanceService}
}

func (c *BalanceController) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int64)

	balance, err := c.balanceService.GetBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, balance)
}
