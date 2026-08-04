package bootstrap

import "github.com/gin-gonic/gin"

func setupAuthRoutes(v1 *gin.RouterGroup, container *Container) {
	auth := v1.Group("/auth")
	{
		auth.POST("/login", container.Middleware.LoginRateLimit, container.Handlers.Auth.Login)
		auth.GET("/captcha", container.Handlers.Auth.GetCaptcha)
		auth.GET("/captcha/required", container.Handlers.Auth.GetCaptchaRequired)
		auth.POST("/login/verify", container.Middleware.LoginRateLimit, container.Handlers.Auth.VerifyLoginOTP)
		auth.POST("/login/resend", container.Middleware.LoginRateLimit, container.Handlers.Auth.ResendOTPCode)
		auth.POST("/google", container.Middleware.LoginRateLimit, container.Handlers.Auth.GoogleLogin)

		// Self-service email password reset (Red Team H6: register only when
		// enabled — disabled means 404, not a misleading 200).
		if container.Config.PasswordReset.Enabled {
			auth.POST("/password-reset/request", container.Middleware.PasswordResetRateLimit, container.Handlers.Auth.RequestPasswordReset)
			auth.POST("/password-reset/confirm", container.Middleware.PasswordResetRateLimit, container.Handlers.Auth.ConfirmPasswordReset)
		}

		// Self-service Zalo-OTP password reset (employee mobile channel).
		// Registered unconditionally; the service hot-checks the admin toggle
		// (zaloconnect.IsEnabled) on every request, so disabling the feature
		// via the admin UI immediately stops dispatch without a 404 that would
		// reveal the toggle state to a prober.
		auth.POST("/zalo-reset/request", container.Middleware.ZaloResetRateLimit, container.Handlers.Auth.RequestZaloReset)
		auth.POST("/zalo-reset/confirm", container.Middleware.ZaloResetConfirmRateLimit, container.Handlers.Auth.ConfirmZaloReset)

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
