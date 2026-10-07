package integration

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/infra/zalo"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// CredentialReader is the read-only slice of zalo.CredentialSource the token
// endpoint needs. zaloconnect.Service implements it against the settings row;
// tests inject a fake.
type CredentialReader interface {
	Get(ctx context.Context) (zalo.Credentials, error)
}

// ZaloTokenHandler serves the chatbot's pull lane for the OA access token.
// Payroll is the sole rotator of the shared Zalo OA token pair: the bot pulls
// the current token here and additionally receives it pushed after every
// rotation and at application startup. The refresh token is never exposed.
type ZaloTokenHandler struct {
	creds CredentialReader
}

// NewZaloTokenHandler constructs the handler.
func NewZaloTokenHandler(creds CredentialReader) *ZaloTokenHandler {
	return &ZaloTokenHandler{creds: creds}
}

// GetToken
// @Summary Return the current Zalo OA access token
// @Description Serve the live access token to the TingTing chatbot so OA replies keep working between rotations. 503 means payroll holds no usable token yet (admin must paste a fresh pair); the body carries the Vietnamese remediation text.
// @Tags integration
// @Security ApiKeyAuth
// @Success 200 {object} response.SuccessResponse
// @Failure 503 {object} response.ErrorResponse
// @Router /integration/zalo/token [get]
func (h *ZaloTokenHandler) GetToken(c *gin.Context) {
	creds, err := h.creds.Get(c.Request.Context())
	status, message := zaloTokenOutcome(creds.AccessToken, err)
	if status != http.StatusOK {
		c.JSON(status, response.ErrorResponse{
			Status:     "error",
			Message:    message,
			HTTPStatus: status,
		})
		return
	}
	response.Success(c, dto.ZaloTokenResponse{AccessToken: creds.AccessToken}, message)
}

// zaloTokenOutcome maps a credential read onto the pull endpoint's HTTP status
// and message. Pure so the mapping is unit-testable. A not-yet-configured
// connection (or an empty stored token) is 503 — retryable once the admin
// pastes a fresh pair — surfaced with ErrNotConfigured's Vietnamese text
// stripped of its "zalo: " log prefix, matching the admin-facing message style.
// The token itself never appears in logs, only in the 200 body.
func zaloTokenOutcome(accessToken string, err error) (int, string) {
	notConfigured := strings.TrimPrefix(zalo.ErrNotConfigured.Error(), "zalo: ")
	switch {
	case errors.Is(err, zalo.ErrNotConfigured):
		return http.StatusServiceUnavailable, notConfigured
	case err != nil:
		return http.StatusInternalServerError, constants.MsgZaloResetStoreDownVN
	case accessToken == "":
		return http.StatusServiceUnavailable, notConfigured
	default:
		return http.StatusOK, constants.MsgIntegrationZaloTokenVN
	}
}
