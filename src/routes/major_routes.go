package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// MajorRoutes กำหนดเส้นทางสำหรับ Major API
func majorRoutes(app *fiber.App) {
	majorRoutes := app.Group("/majors")
	majorRoutes.Get("/", controllers.GetActivitys)         // ดึงผู้ใช้ทั้งหมด
	majorRoutes.Post("/", controllers.CreateActivity)      // สร้างผู้ใช้ใหม่
	majorRoutes.Get("/:id", controllers.GetActivityByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	majorRoutes.Put("/:id", controllers.UpdateActivity)    // อัปเดตข้อมูลผู้ใช้
	majorRoutes.Delete("/:id", controllers.DeleteActivity) // ลบผู้ใช้
}
