// file: src/routes/submission_routes.go
package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

func SubmissionRoutes(router fiber.Router, db *mongo.Database) {
	submissions := router.Group("/submissions")

	// CRUD
	submissions.Post("/", controllers.CreateSubmission)
	submissions.Get("/:id", controllers.GetSubmission)
	submissions.Get("/form/:formId", controllers.GetSubmissionsByForm) // ของเดิม

	submissions.Delete("/:id", controllers.DeleteSubmission)

	// ✅ Analytics (อยู่ใต้ submission ตามที่ frontend เรียกไว้)
	router.Get("/submissions/analytics/forms/:formId/blocks", controllers.GetFormBlocksAnalytics)
	router.Get("/submissions/analytics/forms/:formId/blocks/:blockId", controllers.GetBlockAnalytics)

	router.Get("/forms/:formId/submissions", controllers.GetSubmissionsByForm)
}
