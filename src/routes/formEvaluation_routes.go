package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// FormEvaluationRoutes กำหนดเส้นทางสำหรับ FormEvaluation API
func formEvaluationRoutes(app *fiber.App) {
	formEvaluationRoutes := app.Group("/formEvaluations")
	formEvaluationRoutes.Get("/", controllers.GetActivitys)         // ดึงผู้ใช้ทั้งหมด
	formEvaluationRoutes.Post("/", controllers.CreateActivity)      // สร้างผู้ใช้ใหม่
	formEvaluationRoutes.Get("/:id", controllers.GetActivityByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	formEvaluationRoutes.Put("/:id", controllers.UpdateActivity)    // อัปเดตข้อมูลผู้ใช้
	formEvaluationRoutes.Delete("/:id", controllers.DeleteActivity) // ลบผู้ใช้
}
