package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// EvaluationScoreRoutes กำหนดเส้นทางสำหรับ EvaluationScore API
func evaluationScoreRoutes(app *fiber.App) {
	evaluationScoreRoutes := app.Group("/evaluationScores")
	evaluationScoreRoutes.Get("/", controllers.GetActivitys)         // ดึงผู้ใช้ทั้งหมด
	evaluationScoreRoutes.Post("/", controllers.CreateActivity)      // สร้างผู้ใช้ใหม่
	evaluationScoreRoutes.Get("/:id", controllers.GetActivityByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	evaluationScoreRoutes.Put("/:id", controllers.UpdateActivity)    // อัปเดตข้อมูลผู้ใช้
	evaluationScoreRoutes.Delete("/:id", controllers.DeleteActivity) // ลบผู้ใช้
}
