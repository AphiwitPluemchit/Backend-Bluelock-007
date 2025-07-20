package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// FormEvaluationRoutes กำหนดเส้นทางสำหรับ FormEvaluation API
func formEvaluationRoutes(router fiber.Router) {
	formEvaluationRoutes := router.Group("/formEvaluations")
	// formEvaluationRoutes.Get("/", controllers.GetFormEvaluations)         // ดึงผู้ใช้ทั้งหมด
	formEvaluationRoutes.Post("/", controllers.CreateFormEvaluation)      // สร้างผู้ใช้ใหม่
	formEvaluationRoutes.Get("/:id", controllers.GetFormEvaluationByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	formEvaluationRoutes.Put("/:id", controllers.UpdateFormEvaluation)    // อัปเดตข้อมูลผู้ใช้
	formEvaluationRoutes.Delete("/:id", controllers.DeleteFormEvaluation) // ลบผู้ใช้
}
