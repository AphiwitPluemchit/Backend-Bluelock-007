package routes

import (
	"github.com/gofiber/fiber/v2"
)

func InitRoutes(app *fiber.App) {
	// เรียกใช้ฟังก์ชัน InitUserRoutes และ InitOrderRoutes
	activityRoutes(app)
	activityStateRoutes(app)
	adminRoutes(app)
	checkInOutRoutes(app)

	evaluationScoreRoutes(app)
	foodRoutes(app)
	foodVoteRoutes(app)
	formEvaluationRoutes(app)
	majorRoutes(app)
	skillRoutes(app)
	studentRoutes(app)
	suggestionRoutes(app)

	// Route เช็คว่า API ทำงานอยู่
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("✅ API is running...")
	})
}
