package bootstrap

import "github.com/gin-gonic/gin"

func setupAuthRoutes(v1 *gin.RouterGroup, container *Container) {
	auth := v1.Group("/auth")
	{
		auth.POST("/login", container.Middleware.LoginRateLimit, container.Handlers.Auth.Login)
		auth.POST("/google", container.Handlers.Auth.GoogleLogin)
		auth.POST("/logout", container.Middleware.Auth.Authenticate(), container.Handlers.Auth.Logout)

		auth.GET("/me", container.Middleware.Auth.Authenticate(), container.Handlers.Auth.GetProfile)
		auth.PUT("/me", container.Middleware.Auth.Authenticate(), container.Handlers.Auth.UpdateProfile)
		auth.POST("/change-password", container.Middleware.Auth.Authenticate(), container.Handlers.Auth.ChangePassword)
	}
}

func setupUserRoutes(v1 *gin.RouterGroup, container *Container) {
	users := v1.Group("/users")
	users.Use(container.Middleware.Auth.Authenticate())
	users.Use(container.Middleware.Authorization.Authorize())
	{
		users.POST("", container.Handlers.User.CreateUser)
		users.GET("", container.Handlers.User.ListUsers)
		users.GET("/summary", container.Handlers.User.GetUserSummary)
		users.GET("/:id", container.Handlers.User.GetUser)
		users.GET("/:id/activities", container.Handlers.User.GetUserActivities)
		users.PUT("/:id", container.Handlers.User.UpdateUser)
		users.DELETE("/:id", container.Handlers.User.DeleteUser)
		users.POST("/:id/reset-password", container.Handlers.User.ResetUserPassword)
		users.POST("/reset-first-time-login-password", container.Handlers.User.ResetFirstTimeLoginPasswords)
		users.GET("/reset-first-time-login-password", container.Handlers.User.GetPasswordResetJobStatus)
	}
}
