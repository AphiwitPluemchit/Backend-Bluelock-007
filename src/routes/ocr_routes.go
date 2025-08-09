package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

func ocrRoutes(router fiber.Router) {
	ocr := router.Group("/ocr")
	ocr.Post("/upload", controllers.UploadHandler)
}
