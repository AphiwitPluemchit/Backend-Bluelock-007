package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// MajorRoutes กำหนดเส้นทางสำหรับ Major API
func majorRoutes(app *fiber.App) {
	majorRoutes := app.Group("/majors")
	majorRoutes.Get("/", controllers.GetMajors)         // ดึงผู้ใช้ทั้งหมด
	majorRoutes.Post("/", controllers.CreateMajor)      // สร้างผู้ใช้ใหม่
	majorRoutes.Get("/:id", controllers.GetMajorByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	majorRoutes.Put("/:id", controllers.UpdateMajor)    // อัปเดตข้อมูลผู้ใช้
	majorRoutes.Delete("/:id", controllers.DeleteMajor) // ลบผู้ใช้
}
