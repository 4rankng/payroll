package handlers

import (
	"api-server/internal/transport/http/handlers/bank"

	"api-server/internal/app/services/asset"
)

type BankHandler struct {
	*bank.Handler
}

func NewBankHandler(bankService *asset.BankService) *BankHandler {
	return &BankHandler{
		Handler: bank.NewHandler(bankService),
	}
}
