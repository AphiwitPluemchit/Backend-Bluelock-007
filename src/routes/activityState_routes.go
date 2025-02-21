package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// ActivityStateRoutes กำหนดเส้นทางสำหรับ ActivityState API
func activityStateRoutes(app *fiber.App) {
	activityStateRoutes := app.Group("/activityStates")
	activityStateRoutes.Get("/", controllers.GetActivityStates)         // ดึงผู้ใช้ทั้งหมด
	activityStateRoutes.Post("/", controllers.CreateActivityState)      // สร้างผู้ใช้ใหม่
	activityStateRoutes.Get("/:id", controllers.GetActivityStateByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	activityStateRoutes.Put("/:id", controllers.UpdateActivityState)    // อัปเดตข้อมูลผู้ใช้
	activityStateRoutes.Delete("/:id", controllers.DeleteActivityState) // ลบผู้ใช้
}
