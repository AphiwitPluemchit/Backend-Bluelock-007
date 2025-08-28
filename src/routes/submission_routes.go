// file: src/routes/submission_routes.go
package routes

import (
	submissionService "Backend-Bluelock-007/src/services/submission"
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)


func SubmissionRoutes(router fiber.Router, db *mongo.Database) {
	// สร้าง service และ controller
	svc := submissionService.NewSubmissionService(db)
	ctrl := controllers.NewSubmissionController(svc)

	submissions := router.Group("/submissions")

	// Create
	submissions.Post("/", ctrl.CreateSubmission)

	// Read
	submissions.Get("/:id", ctrl.GetSubmission)               // GET /submissions/:id
	submissions.Get("/form/:formId", ctrl.GetSubmissionsByForm) // GET /submissions/form/:formId

	// Delete
	submissions.Delete("/:id", ctrl.DeleteSubmission)
}
