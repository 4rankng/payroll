package admin

import (
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/mcpapikey"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// MCPAPIKeyHandler exposes admin-only endpoints for managing machine API keys
// used by the MCP agent surface (/api/v1/mcp). All routes live behind the
// admin authorization middleware.
type MCPAPIKeyHandler struct {
	service *mcpapikey.Service
}

// NewMCPAPIKeyHandler constructs the handler.
func NewMCPAPIKeyHandler(service *mcpapikey.Service) *MCPAPIKeyHandler {
	return &MCPAPIKeyHandler{service: service}
}

// toMCPAPIKeyResponse maps a domain key to its DTO. plaintext is non-empty
// only on the create response.
func toMCPAPIKeyResponse(key *domain.MCPAPIKey, plaintext string) dto.MCPAPIKeyResponse {
	return dto.MCPAPIKeyResponse{
		ID:         key.ID,
		Name:       key.Name,
		Scope:      key.Scope,
		Key:        plaintext,
		KeyPrefix:  key.KeyPrefix,
		CreatedBy:  key.CreatedBy,
		LastUsedAt: key.LastUsedAt,
		RevokedAt:  key.RevokedAt,
		CreatedAt:  key.CreatedAt,
	}
}

// Create
// @Summary Create an MCP API key
// @Description Mint a new MCP machine API key. The plaintext key is returned ONCE.
// @Tags admin,mcp-api-keys
// @Security Bearer
// @Param body body dto.CreateMCPAPIKeyRequest true "Key name and scope"
// @Success 201 {object} response.SuccessResponse
// @Router /admin/mcp-api-keys [post]
func (h *MCPAPIKeyHandler) Create(c *gin.Context) {
	var req dto.CreateMCPAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	actorID, _ := c.Get(constants.CtxUserID)
	createdBy, _ := actorID.(uint)

	key, plaintext, err := h.service.Create(c.Request.Context(), req.Name, req.Scope, createdBy)
	if err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	response.SuccessCreated(c, toMCPAPIKeyResponse(key, plaintext), constants.MsgAPIKeyCreatedVN)
}

// List
// @Summary List MCP API keys
// @Description List every MCP API key (revoked included) with masked prefixes.
// @Tags admin,mcp-api-keys
// @Security Bearer
// @Success 200 {object} response.SuccessResponse
// @Router /admin/mcp-api-keys [get]
func (h *MCPAPIKeyHandler) List(c *gin.Context) {
	keys, err := h.service.List(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	out := make([]dto.MCPAPIKeyResponse, 0, len(keys))
	for _, k := range keys {
		out = append(out, toMCPAPIKeyResponse(k, ""))
	}
	response.Success(c, out, constants.MsgSuccessVN)
}

// Revoke
// @Summary Revoke an MCP API key
// @Description Revoke a key; it can no longer authenticate MCP calls.
// @Tags admin,mcp-api-keys
// @Security Bearer
// @Param id path int true "MCP API key ID"
// @Success 200 {object} response.SuccessResponse
// @Router /admin/mcp-api-keys/{id} [delete]
func (h *MCPAPIKeyHandler) Revoke(c *gin.Context) {
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
