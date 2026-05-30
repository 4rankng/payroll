package push

import (
	"context"
	"net/http"

	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	pushService PushServiceAPI
}

type PushServiceAPI interface {
	Subscribe(ctx context.Context, userID uint, endpoint, p256dh, auth, deviceType string) error
	Unsubscribe(ctx context.Context, userID uint, endpoint string) error
	GetVAPIDPublicKey() string
}

func NewHandler(pushService PushServiceAPI) *Handler {
	return &Handler{pushService: pushService}
}

type SubscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
	Keys     struct {
		P256DH string `json:"p256dh" binding:"required"`
		Auth   string `json:"auth" binding:"required"`
	} `json:"keys" binding:"required"`
	DeviceType string `json:"deviceType"`
}

// GetVAPIDKey returns the VAPID public key
func (h *Handler) GetVAPIDKey(c *gin.Context) {
	key := h.pushService.GetVAPIDPublicKey()
	if key == "" {
		response.InternalServerError(c, "Push notifications not configured")
		return
	}
	c.JSON(http.StatusOK, gin.H{"publicKey": key})
}

// Subscribe registers a push subscription for the authenticated user
func (h *Handler) Subscribe(c *gin.Context) {
	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp liệu")
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		response.Forbidden(c, constants.MsgInvalidUserIDVN)
		return
	}

	deviceType := req.DeviceType
	if deviceType == "" {
		deviceType = "web"
	}

	if err := h.pushService.Subscribe(c.Request.Context(), uid, req.Endpoint, req.Keys.P256DH, req.Keys.Auth, deviceType); err != nil {
		response.InternalServerError(c, "Không thể đăng ký thông báo")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đăng ký thông báo thành công"})
}

// Unsubscribe removes a push subscription
func (h *Handler) Unsubscribe(c *gin.Context) {
	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		response.Forbidden(c, constants.MsgInvalidUserIDVN)
		return
	}

	if err := h.pushService.Unsubscribe(c.Request.Context(), uid, req.Endpoint); err != nil {
		response.InternalServerError(c, "Không thể hủy đăng ký thông báo")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã hủy đăng ký thông báo"})
}
