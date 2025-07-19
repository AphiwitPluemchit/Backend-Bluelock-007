package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// AuthRoutes กำหนด route สำหรับ auth (login/logout/register)
func authRoutes(router fiber.Router) {
	auth := router.Group("/auth")

	auth.Post("/login", controllers.LoginUser) // 🔐 login
}
