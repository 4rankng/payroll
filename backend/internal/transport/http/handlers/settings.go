package handlers

import (
	"api-server/internal/transport/http/handlers/settings"

	"api-server/internal/app/services/config"
)

type SettingsHandler struct {
	*settings.Handler
}

func NewSettingsHandler(settingsService *config.SettingsService) *SettingsHandler {
	return &SettingsHandler{
		Handler: settings.NewHandler(settingsService),
	}
}
