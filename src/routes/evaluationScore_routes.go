package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// EvaluationScoreRoutes กำหนดเส้นทางสำหรับ EvaluationScore API
func evaluationScoreRoutes(router fiber.Router) {
	evaluationScoreRoutes := router.Group("/evaluationScores")
	// evaluationScoreRoutes.Get("/", controllers.GetEvaluationScores)         // ดึงผู้ใช้ทั้งหมด
	evaluationScoreRoutes.Post("/", controllers.CreateEvaluationScore)      // สร้างผู้ใช้ใหม่
	evaluationScoreRoutes.Get("/:id", controllers.GetEvaluationScoreByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	evaluationScoreRoutes.Put("/:id", controllers.UpdateEvaluationScore)    // อัปเดตข้อมูลผู้ใช้
	evaluationScoreRoutes.Delete("/:id", controllers.DeleteEvaluationScore) // ลบผู้ใช้
}
