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
	forms.Get("/", controllers.GetForms)         // Get all forms with pagination
	forms.Get("/:id", controllers.GetFormByID)   // Get a specific form with questions
	forms.Delete("/:id", controllers.DeleteForm) // Delete a form
	// Form submission routes
	forms.Post("/:id/submissions", controllers.SubmitForm)        // Submit answers to a form
	forms.Get("/:id/submissions", controllers.GetFormSubmissions) // Get all submissions for a form
}
