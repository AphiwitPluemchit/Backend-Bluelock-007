package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// ActivityItemRoutes กำหนดเส้นทางสำหรับ ActivityItem API
func activityItemRoutes(app *fiber.App) {
	activityItemRoutes := app.Group("/activityItems")
	activityItemRoutes.Get("/", controllers.GetActivityItems)         // ดึงผู้ใช้ทั้งหมด
	activityItemRoutes.Post("/", controllers.CreateActivityItem)      // สร้างผู้ใช้ใหม่
	activityItemRoutes.Get("/:id", controllers.GetActivityItemByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	activityItemRoutes.Put("/:id", controllers.UpdateActivityItem)    // อัปเดตข้อมูลผู้ใช้
	activityItemRoutes.Delete("/:id", controllers.DeleteActivityItem) // ลบผู้ใช้
}
