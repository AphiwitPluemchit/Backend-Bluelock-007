package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// CheckInOutRoutes กำหนดเส้นทางสำหรับ CheckInOut API
func checkInOutRoutes(app *fiber.App) {
	checkInOutRoutes := app.Group("/checkInOuts")
	// checkInOutRoutes.Get("/", controllers.GetCheckInOuts)         // ดึงผู้ใช้ทั้งหมด
	checkInOutRoutes.Post("/", controllers.CreateCheckInOut)      // สร้างผู้ใช้ใหม่
	checkInOutRoutes.Get("/:id", controllers.GetCheckInOutByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	checkInOutRoutes.Put("/:id", controllers.UpdateCheckInOut)    // อัปเดตข้อมูลผู้ใช้
	checkInOutRoutes.Delete("/:id", controllers.DeleteCheckInOut) // ลบผู้ใช้
}
