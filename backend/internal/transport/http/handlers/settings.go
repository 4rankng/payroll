package handlers

import (
	"api-server/internal/transport/http/handlers/settings"

	"api-server/internal/app/services/config"
	llmsvc "api-server/internal/app/services/llm"
)

type SettingsHandler struct {
	*settings.Handler
}

func NewSettingsHandler(settingsService *config.SettingsService, mimoService *llmsvc.Service) *SettingsHandler {
	return &SettingsHandler{
		Handler: settings.NewHandler(settingsService, mimoService),
	}
}
