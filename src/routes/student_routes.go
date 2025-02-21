package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// StudentRoutes กำหนดเส้นทางสำหรับ Student API
func studentRoutes(app *fiber.App) {
	studentRoutes := app.Group("/students")
	studentRoutes.Get("/", controllers.GetActivitys)         // ดึงผู้ใช้ทั้งหมด
	studentRoutes.Post("/", controllers.CreateActivity)      // สร้างผู้ใช้ใหม่
	studentRoutes.Get("/:id", controllers.GetActivityByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	studentRoutes.Put("/:id", controllers.UpdateActivity)    // อัปเดตข้อมูลผู้ใช้
	studentRoutes.Delete("/:id", controllers.DeleteActivity) // ลบผู้ใช้
}
