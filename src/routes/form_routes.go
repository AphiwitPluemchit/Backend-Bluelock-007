package routes

import (
	"Backend-Bluelock-007/src/controllers"
	// "Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// FormRoutes กำหนด route สำหรับ form management
func formRoutes(router fiber.Router) {
	forms := router.Group("/forms")

	forms.Post("/", controllers.CreateForm)      
	forms.Get("/", controllers.GetAllForms)
	forms.Get("/:id", controllers.GetFormByID)
	forms.Delete("/:id", controllers.DeleteFormByid)
	forms.Put("/:id", controllers.UpdateForm)
	forms.Patch("/:id", controllers.UpdateForm)
}
