package admin

import (
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/opsapikey"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// OpsAPIKeyHandler exposes admin-only endpoints for managing machine API
// keys used by the payroll-ops-mcp agent surface (pk_ops_*). All routes live
// behind the admin authorization middleware; the scope toggle additionally
// requires an OTP-verified session (route-level gate).
type OpsAPIKeyHandler struct {
	service *opsapikey.Service
}

// NewOpsAPIKeyHandler constructs the handler.
func NewOpsAPIKeyHandler(service *opsapikey.Service) *OpsAPIKeyHandler {
	return &OpsAPIKeyHandler{service: service}
}

// toOpsAPIKeyResponse maps a domain key to its DTO. plaintext is non-empty
// only on the create response.
func toOpsAPIKeyResponse(key *domain.OpsAPIKey, plaintext string, now time.Time) dto.OpsAPIKeyResponse {
	return dto.OpsAPIKeyResponse{
		ID:                   key.ID,
		Name:                 key.Name,
		Scope:                key.Scope,
		EffectiveScope:       key.EffectiveScope(now),
		WriteWindowExpiresAt: key.WriteWindowExpiresAt,
		Key:                  plaintext,
		KeyPrefix:            key.KeyPrefix,
		CreatedBy:            key.CreatedBy,
		LastUsedAt:           key.LastUsedAt,
		RevokedAt:            key.RevokedAt,
		CreatedAt:            key.CreatedAt,
	}
}

// Create
// @Summary Create an ops API key
// @Description Mint a new pk_ops_* machine key for one agent. The plaintext key is returned ONCE.
// @Tags admin,ops-api-keys
// @Security Bearer
// @Param body body dto.CreateOpsAPIKeyRequest true "Key name, optional scope + write window"
// @Success 201 {object} response.SuccessResponse
// @Router /admin/ops-api-keys [post]
func (h *OpsAPIKeyHandler) Create(c *gin.Context) {
	var req dto.CreateOpsAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	actorID, _ := c.Get(constants.CtxUserID)
	createdBy, _ := actorID.(uint)

	key, plaintext, err := h.service.Create(c.Request.Context(), req.Name, req.Scope, createdBy, req.WriteWindowHours)
	if err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	response.SuccessCreated(c, toOpsAPIKeyResponse(key, plaintext, clock.Now()), constants.MsgAPIKeyCreatedVN)
}

// List
// @Summary List ops API keys
// @Description List every ops key (revoked included) with effective scope and masked prefixes.
// @Tags admin,ops-api-keys
// @Security Bearer
// @Success 200 {object} response.SuccessResponse
// @Router /admin/ops-api-keys [get]
func (h *OpsAPIKeyHandler) List(c *gin.Context) {
	keys, err := h.service.List(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}
	now := clock.Now()
	out := make([]dto.OpsAPIKeyResponse, 0, len(keys))
	for _, k := range keys {
		out = append(out, toOpsAPIKeyResponse(k, "", now))
	}
	response.Success(c, out, constants.MsgSuccessVN)
}

// Revoke
// @Summary Revoke an ops API key
// @Description Revoke a key; it can no longer authenticate ops calls.
// @Tags admin,ops-api-keys
// @Security Bearer
// @Param id path int true "Ops API key ID"
// @Success 200 {object} response.SuccessResponse
// @Router /admin/ops-api-keys/{id} [delete]
func (h *OpsAPIKeyHandler) Revoke(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	actorID, _ := c.Get(constants.CtxUserID)
	actor, _ := actorID.(uint)

	if err := h.service.Revoke(c.Request.Context(), uint(id), actor); err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgAPIKeyNotFoundVN)
			return
		}
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}
	response.Success(c, nil, constants.MsgAPIKeyRevokedVN)
}

// SetScope
// @Summary Toggle an ops key scope
// @Description Switch read_only <-> read_write. Requires an OTP-verified session (2FA re-entry); the change is audit-logged and applies to the very next call.
// @Tags admin,ops-api-keys
// @Security Bearer
// @Param id path int true "Ops API key ID"
// @Param body body dto.SetOpsAPIKeyScopeRequest true "New scope + optional write window hours"
// @Success 200 {object} response.SuccessResponse
// @Router /admin/ops-api-keys/{id}/scope [put]
func (h *OpsAPIKeyHandler) SetScope(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	var req dto.SetOpsAPIKeyScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	actorID, _ := c.Get(constants.CtxUserID)
	actor, _ := actorID.(uint)

	key, err := h.service.SetScope(c.Request.Context(), uint(id), req.Scope, req.WriteWindowHours, actor)
	if err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgAPIKeyNotFoundVN)
			return
		}
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}
	response.Success(c, toOpsAPIKeyResponse(key, "", clock.Now()), "Đã cập nhật phạm vi khoá API")
}

// SelfInfo
// @Summary Ops key self-introspection
// @Description What the authenticated pk_ops_* key may do RIGHT NOW. Used by the payroll-ops-mcp server at startup and per call to decide which tools to expose. Only reachable through an ops key.
// @Tags ops,key-info
// @Success 200 {object} response.SuccessResponse
// @Router /ops/key-info [get]
func (h *OpsAPIKeyHandler) SelfInfo(c *gin.Context) {
	keyIDRaw, exists := c.Get(constants.CtxOpsKeyID)
	if !exists {
		response.Forbidden(c, constants.MsgOpsKeyWrongSurfaceVN)
		return
	}
	keyID, _ := keyIDRaw.(uint)
	key, err := h.service.GetByID(c.Request.Context(), keyID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.Unauthorized(c, constants.MsgOpsKeyInvalidVN)
			return
		}
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}
	response.Success(c, dto.OpsKeySelfInfo{
		KeyID:          key.ID,
		AgentName:      key.Name,
		Scope:          key.Scope,
		EffectiveScope: key.EffectiveScope(clock.Now()),
	}, constants.MsgSuccessVN)
}
