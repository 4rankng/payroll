package admin

import (
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/apikey"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// APIKeyHandler exposes admin-only endpoints for managing machine API keys used
// by the external chatbot integration API. All routes live behind the admin
// authorization middleware.
type APIKeyHandler struct {
	service *apikey.Service
}

// NewAPIKeyHandler constructs the handler.
func NewAPIKeyHandler(service *apikey.Service) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

// Create
// @Summary Create an API key
// @Description Mint a new machine API key. The plaintext key is returned ONCE.
// @Tags admin,api-keys
// @Security Bearer
// @Param body body dto.CreateAPIKeyRequest true "Key name"
// @Success 201 {object} response.SuccessResponse
// @Router /admin/api-keys [post]
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req dto.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	actorID, _ := c.Get(constants.CtxUserID)
	createdBy, _ := actorID.(uint)

	key, plaintext, err := h.service.Create(c.Request.Context(), req.Name, createdBy)
	if err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	response.SuccessCreated(c, toAPIKeyResponse(key, plaintext), constants.MsgAPIKeyCreatedVN)
}

// List
// @Summary List API keys
// @Description List every API key (revoked included) with masked prefixes.
// @Tags admin,api-keys
// @Security Bearer
// @Success 200 {object} response.SuccessResponse
// @Router /admin/api-keys [get]
func (h *APIKeyHandler) List(c *gin.Context) {
	keys, err := h.service.List(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	out := make([]dto.APIKeyResponse, 0, len(keys))
	for _, k := range keys {
		out = append(out, toAPIKeyResponse(k, ""))
	}
	response.Success(c, out, constants.MsgSuccessVN)
}

// Revoke
// @Summary Revoke an API key
// @Description Revoke a key; it can no longer authenticate integration calls.
// @Tags admin,api-keys
// @Security Bearer
// @Param id path int true "API key ID"
// @Success 200 {object} response.SuccessResponse
// @Router /admin/api-keys/{id} [delete]
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	if err := h.service.Revoke(c.Request.Context(), uint(id)); err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgAPIKeyNotFoundVN)
			return
		}
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	response.Success(c, nil, constants.MsgAPIKeyRevokedVN)
}

// toAPIKeyResponse maps a domain key to its DTO. plaintext is non-empty only
// on the create response.
func toAPIKeyResponse(key *domain.APIKey, plaintext string) dto.APIKeyResponse {
	return dto.APIKeyResponse{
		ID:         key.ID,
		Name:       key.Name,
		Key:        plaintext,
		KeyPrefix:  key.KeyPrefix,
		CreatedBy:  key.CreatedBy,
		LastUsedAt: key.LastUsedAt,
		RevokedAt:  key.RevokedAt,
		CreatedAt:  key.CreatedAt,
	}
}
