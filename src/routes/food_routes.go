package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// FoodRoutes กำหนดเส้นทางสำหรับ Food API
func foodRoutes(app *fiber.App) {
	foodRoutes := app.Group("/foods")
	foodRoutes.Get("/", controllers.GetActivitys)         // ดึงผู้ใช้ทั้งหมด
	foodRoutes.Post("/", controllers.CreateActivity)      // สร้างผู้ใช้ใหม่
	foodRoutes.Get("/:id", controllers.GetActivityByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	foodRoutes.Put("/:id", controllers.UpdateActivity)    // อัปเดตข้อมูลผู้ใช้
	foodRoutes.Delete("/:id", controllers.DeleteActivity) // ลบผู้ใช้
}
