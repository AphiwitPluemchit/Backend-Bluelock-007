package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

func certificateRoutes(router fiber.Router) {
	certificate := router.Group("/certificates")
	certificate.Get("/url-verify", controllers.VerifyURL)
}
