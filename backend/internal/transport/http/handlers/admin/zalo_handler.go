package admin

import (
	"api-server/internal/app/services/zaloconnect"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ZaloHandler exposes admin-only endpoints for managing the Zalo OA connection
// (credentials, runtime toggle, forced refresh, test send).
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
// (the UI sends empty for password fields the admin did not retype). The admin
// pastes all four OA fields directly from the Zalo OA Console; there is no
// OAuth authorization-code flow.
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
// @Description Save app_id/secret/template and the manually-pasted tokens.
// @Description Each field is applied independently — empty fields mean "keep existing"
// @Description so the admin can re-paste one value without clobbering the others.
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
// @Summary Force a Zalo token refresh (validates connection)
// @Description Used by the admin "Kiểm tra kết nối" button — validates
// @Description App ID + Secret + Refresh Token against Zalo's token endpoint
// @Description without sending a ZNS message (useful when the template is
// @Description still pending Zalo approval).
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

// zaloTestSendDTO is the body for POST /admin/zalo/test. phone is required;
// template_id defaults to the OTP template (617976); template_data defaults to
// sample values for the OTP template. For any other template the admin must
// supply template_data matching the template's declared params.
type zaloTestSendDTO struct {
	Phone        string            `json:"phone" binding:"required"`
	TemplateID   string            `json:"template_id"`
	TemplateData map[string]string `json:"template_data"`
}

// TestSend
// @Summary Send a test ZNS message
// @Description Fires one ZNS template message to verify the stored tokens work.
// @Description Defaults to the OTP template (617976) with sample data.
// @Description Does NOT touch the password-reset flow — no OTP stored in Redis.
// @Tags admin,zalo
// @Security Bearer
// @Param body body zaloTestSendDTO true "Phone + optional template/data"
// @Success 200 {object} response.SuccessResponse
// @Router /admin/zalo/test [post]
func (h *ZaloHandler) TestSend(c *gin.Context) {
	var req zaloTestSendDTO
	if !helpers.BindJSON(c, &req) {
		return
	}
	res, err := h.svc.TestSend(c.Request.Context(), req.Phone, req.TemplateID, req.TemplateData)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, res, "Đã gửi tin thử")
}
