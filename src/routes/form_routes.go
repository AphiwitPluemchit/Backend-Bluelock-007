package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// FormRoutes กำหนด route สำหรับ form management
func formRoutes(router fiber.Router) {
	forms := router.Group("/forms")
	forms.Use(middleware.AuthJWT)
	forms.Post("/", controllers.CreateForm)      // Create a new form
}
