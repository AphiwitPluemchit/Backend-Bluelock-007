package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// AuthRoutes กำหนด route สำหรับ auth (login/logout/register)
func authRoutes(app *fiber.App) {
	auth := app.Group("/auth")

	auth.Post("/login", controllers.LoginUser) // 🔐 login
}
