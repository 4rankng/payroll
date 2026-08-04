package admin

import (
	"api-server/internal/app/services/zaloconnect"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ZaloHandler exposes admin-only endpoints for managing the Zalo OA connection
// (credentials, OAuth connect flow, runtime toggle, forced refresh).
// All routes live behind the admin authorization middleware.
type ZaloHandler struct {
	svc *zaloconnect.Service
}

// NewZaloHandler constructs the handler.
func NewZaloHandler(svc *zaloconnect.Service) *ZaloHandler {
	return &ZaloHandler{svc: svc}
}

// zaloCredentialsDTO is the body for PUT /admin/zalo/credentials.
// SecretKey/AccessToken/RefreshToken are optional — empty means "keep existing"
// (the UI sends empty for password fields the admin did not retype). This lets
// the admin paste tokens manually for localhost/dev (no OAuth callback URL
// reachable from the OA) OR run the OAuth connect flow for production.
type zaloCredentialsDTO struct {
	AppID        string `json:"app_id" binding:"required"`
	SecretKey    string `json:"secret_key"`
	TemplateID   string `json:"template_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// zaloEnabledDTO is the body for PUT /admin/zalo/enabled.
type zaloEnabledDTO struct {
	Enabled bool `json:"enabled"`
}

// zaloOAuthStartDTO is the response for POST /admin/zalo/oauth/start.
type zaloOAuthStartDTO struct {
	RedirectURL string `json:"redirect_url"`
}

// zaloOAuthCallbackQuery is the query string Zalo appends to the redirect.
type zaloOAuthCallbackQuery struct {
	Code  string `form:"code" form:"code"`
	State string `form:"state" form:"state"`
}

// zaloOAuthCallbackDTO is the JSON body the SPA POSTs after receiving the
// Zalo redirect. The SPA extracts code+state from the URL query (Zalo's
// redirect lands on a frontend route), then calls this endpoint via axios —
// which attaches the admin's Bearer JWT. A direct browser redirect to this
// API endpoint would NOT carry the JWT (the SPA uses header-based auth, not
// cookies), so the SPA intermediary is required.
type zaloOAuthCallbackDTO struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// GetStatus
// @Summary Get Zalo connection status
// @Description Masked view of the Zalo OA connection (never returns secret_key/tokens).
// @Tags admin,zalo
// @Security Bearer
// @Success 200 {object} response.SuccessResponse
// @Router /admin/zalo [get]
func (h *ZaloHandler) GetStatus(c *gin.Context) {
	status, err := h.svc.GetStatus(c.Request.Context())
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, status, "ok")
}

// SaveCredentials
// @Summary Save Zalo OA credentials
// @Description Save app_id/secret/template and optional manually-pasted tokens.
// @Description Tokens are preserved when app_id is unchanged; cleared when app_id rotates.
// @Tags admin,zalo
// @Security Bearer
// @Param body body zaloCredentialsDTO true "Credentials"
// @Success 200 {object} response.SuccessResponse
// @Router /admin/zalo/credentials [put]
func (h *ZaloHandler) SaveCredentials(c *gin.Context) {
	var req zaloCredentialsDTO
	if !helpers.BindJSON(c, &req) {
		return
	}
	if err := h.svc.SaveCredentials(c.Request.Context(), zaloconnect.SaveCredentialsInput{
		AppID:        req.AppID,
		SecretKey:    req.SecretKey,
		TemplateID:   req.TemplateID,
		AccessToken:  req.AccessToken,
		RefreshToken: req.RefreshToken,
	}); err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.SuccessEmpty(c, "Đã lưu thông tin kết nối Zalo")
}

// StartOAuth
// @Summary Start Zalo OAuth connect flow
// @Description Mints a single-use state and returns the Zalo permission URL.
// @Tags admin,zalo
// @Security Bearer
// @Success 200 {object} response.SuccessResponse
// @Router /admin/zalo/oauth/start [post]
func (h *ZaloHandler) StartOAuth(c *gin.Context) {
	redirectURL, err := h.svc.StartOAuth(c.Request.Context())
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, zaloOAuthStartDTO{RedirectURL: redirectURL}, "ok")
}

// OAuthCallback
// @Summary Complete Zalo OAuth connect (exchange code for tokens)
// @Description Admin-authenticated. The SPA calls this after receiving Zalo's
// @Description redirect (which lands on a frontend route carrying code+state in
// @Description the URL query). The SPA extracts them and POSTs here via axios
// @Description (which attaches the admin's JWT). Validates the single-use state,
// @Description exchanges the code, persists tokens.
// @Tags admin,zalo
// @Security Bearer
// @Param body body zaloOAuthCallbackDTO true "Authorization code + state"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Router /admin/zalo/oauth/callback [post]
func (h *ZaloHandler) OAuthCallback(c *gin.Context) {
	var req zaloOAuthCallbackDTO
	if !helpers.BindJSON(c, &req) {
		return
	}
	if err := h.svc.HandleOAuthCallback(c.Request.Context(), req.Code, req.State); err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.SuccessEmpty(c, "Đã kết nối Zalo thành công")
}

// SetEnabled
// @Summary Toggle Zalo OTP feature on/off
// @Description Hot toggle — takes effect on the next request, no redeploy.
// @Tags admin,zalo
// @Security Bearer
// @Param body body zaloEnabledDTO true "Enabled flag"
// @Success 200 {object} response.SuccessResponse
// @Router /admin/zalo/enabled [put]
func (h *ZaloHandler) SetEnabled(c *gin.Context) {
	var req zaloEnabledDTO
	if !helpers.BindJSON(c, &req) {
		return
	}
	if err := h.svc.SetEnabled(c.Request.Context(), req.Enabled); err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.SuccessEmpty(c, "Đã cập nhật trạng thái bật Zalo OTP")
}

// RefreshNow
// @Summary Force a Zalo token refresh
// @Description Manual refresh for debugging ("Làm mới token").
// @Tags admin,zalo
// @Security Bearer
// @Success 200 {object} response.SuccessResponse
// @Router /admin/zalo/refresh [post]
func (h *ZaloHandler) RefreshNow(c *gin.Context) {
	if err := h.svc.RefreshNow(c.Request.Context()); err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.SuccessEmpty(c, "Đã làm mới token Zalo")
}
