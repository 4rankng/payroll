package handlers

import (
	"api-server/internal/transport/http/handlers/lender"

	"api-server/internal/app/services/loan"
)

type LenderHandler struct {
	*lender.Handler
}

func NewLenderHandler(lenderService *loan.LenderService) *LenderHandler {
	return &LenderHandler{
		Handler: lender.NewHandler(lenderService),
	}
}
