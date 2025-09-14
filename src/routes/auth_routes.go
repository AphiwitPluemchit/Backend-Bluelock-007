package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// AuthRoutes กำหนด route สำหรับ auth (login/logout/register)
func authRoutes(router fiber.Router) {
	auth := router.Group("/auth")
	auth.Post("/login", controllers.LoginUser)                       // 🔐 login (no auth required)
	auth.Post("/logout", middleware.AuthJWT, controllers.LogoutUser) // 🔐 logout (requires JWT auth)
	
	// Google OAuth routes
	auth.Get("/google", controllers.GoogleLogin)           // 🔐 start Google OAuth flow
	auth.Get("/google/redirect", controllers.GoogleCallback) // 🔐 Google OAuth callback
}
