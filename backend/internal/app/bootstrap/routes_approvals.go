package bootstrap

import (
	"github.com/gin-gonic/gin"
)

// setupApprovalRoutes mounts the approval-request engine API.
//
// RBAC (configs/casbin_policy.csv): partner may list-own and file; decision
// paths carry explicit deny rows for every non-admin role (keyMatch2 defense
// in depth), and the service re-checks role on every decision behind Casbin.
func setupApprovalRoutes(protected *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.Approval == nil {
		return
	}
	g := protected.Group("/approvals")
	{
		g.GET("", container.Handlers.Approval.List)
		g.POST("", container.Handlers.Approval.File)
		g.GET("/:id", container.Handlers.Approval.Get)
		g.POST("/:id/approve", container.Handlers.Approval.Approve)
		g.POST("/:id/reject", container.Handlers.Approval.Reject)
		g.POST("/:id/cancel", container.Handlers.Approval.Cancel)
	}
}
