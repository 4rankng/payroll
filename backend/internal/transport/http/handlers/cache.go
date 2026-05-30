package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"
)

type CacheHandler struct {
	cacheService domain.CacheServiceUseCase
	logger       *slog.Logger
}

func NewCacheHandler(cacheService domain.CacheServiceUseCase, logger *slog.Logger) *CacheHandler {
	return &CacheHandler{
		cacheService: cacheService,
		logger:       logger,
	}
}

func (h *CacheHandler) RegisterRoutes(router *gin.Engine) {
	cacheRoutes := router.Group("/api/v1/cache")
	{
		cacheRoutes.DELETE("", h.DeleteAllCache)
	}
}

// DeleteAllCache godoc
// @Summary Delete all cache
// @Description Delete all cache from the system
// @Tags Tools
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/cache [delete]
func (h *CacheHandler) DeleteAllCache(c *gin.Context) {
	userRole, ok := c.Get("user_role")
	if !ok {
		response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
		return
	}

	if userRole != string(domain.RoleAdmin) {
		response.Forbidden(c, constants.MsgOnlyAdminCanSendNotificationsVN)
		return
	}

	if err := h.cacheService.FlushAll(c.Request.Context()); err != nil {
		h.logger.Error("Failed to delete all cache", "error", err)
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	response.Success(c, nil, "All cache deleted successfully")
}
