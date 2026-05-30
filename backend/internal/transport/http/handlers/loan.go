package handlers

import (
	"api-server/internal/app/services/loan"
	"api-server/internal/pkg/clock"
	loanHandler "api-server/internal/transport/http/handlers/loan"
)

type LoanHandler struct {
	*loanHandler.Handler
}

func NewLoanHandler(loanService *loan.LoanService, clk clock.Clock) *LoanHandler {
	return &LoanHandler{
		Handler: loanHandler.NewHandler(loanService, clk),
	}
}
