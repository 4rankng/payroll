package bootstrap

import (
	"github.com/gin-gonic/gin"
)

// setupChatRoutes mounts the MiMo assistant chat surface (Admin + Partner).
//
// RBAC: the Casbin rows grant admin (wildcard) and partner; adv_partner,
// accountant and employee carry explicit deny rows (defense in depth —
// adv_partner reuses the admin layout and must not see the assistant). The
// handler and the service additionally re-check the JWT role, and the service
// enforces per-record conversation ownership on every access (Casbin is
// role×path only).
func setupChatRoutes(protected *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.Chat == nil {
		return
	}
	g := protected.Group("/chat")
	{
		g.POST("/conversations", container.Handlers.Chat.CreateConversation)
		g.GET("/conversations", container.Handlers.Chat.ListConversations)
		g.GET("/conversations/:id", container.Handlers.Chat.GetConversation)
		g.POST("/conversations/:id/messages", container.Handlers.Chat.SendMessage)
		g.POST("/attachments", container.Handlers.Chat.UploadAttachment)
	}
}
