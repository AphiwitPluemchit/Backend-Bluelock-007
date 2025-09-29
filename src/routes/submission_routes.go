// file: src/routes/submission_routes.go
package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

func SubmissionRoutes(router fiber.Router, _ *mongo.Database) {
	// กลุ่ม /submissions (CRUD เดิม)
	submissions := router.Group("/submissions")

	// Create
	submissions.Post("/", controllers.CreateSubmission)

	// Read
	submissions.Get("/:id", controllers.GetSubmission)                 // GET /submissions/:id
	submissions.Get("/form/:formId", controllers.GetSubmissionsByForm) // GET /submissions/form/:formId (legacy)


	router.Get("/forms/:formId/submissions", controllers.GetSubmissionsByFormWithQuery)

	// ✅ Analytics (แนะนำพาธระดับ root ให้ตรงกับ frontend)
	router.Get("/analytics/forms/:formId/blocks", controllers.GetFormBlocksAnalytics)
	router.Get("/analytics/forms/:formId/blocks/:blockId", controllers.GetBlockAnalytics)
}
