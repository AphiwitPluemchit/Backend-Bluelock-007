// file: src/routes/submission_routes.go
package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

func SubmissionRoutes(router fiber.Router, db *mongo.Database) {
	// สร้าง service และ controller
	// svc := submissionService.NewSubmissionService(db)
	// ctrl := controllers.NewSubmissionController(svc)

	submissions := router.Group("/submissions")

	// Create
	submissions.Post("/", controllers.CreateSubmission)

	// Read
	submissions.Get("/:id", controllers.GetSubmission)                 // GET /submissions/:id
	submissions.Get("/form/:formId", controllers.GetSubmissionsByForm) // GET /submissions/form/:formId

	// Delete
	submissions.Delete("/:id", controllers.DeleteSubmission)
}
