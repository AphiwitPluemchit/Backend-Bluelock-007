package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// ActivityRoutes กำหนดเส้นทางสำหรับ Activity API
func activityRoutes(app *fiber.App) {
	activityRoutes := app.Group("/activitys")
	activityRoutes.Get("/", controllers.GetAllActivities)     // ดึงผู้ใช้ทั้งหมด
	activityRoutes.Post("/", controllers.CreateActivity)      // สร้างผู้ใช้ใหม่
	activityRoutes.Get("/:id", controllers.GetActivityByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	activityRoutes.Put("/:id", controllers.UpdateActivity)    // อัปเดตข้อมูลผู้ใช้
	activityRoutes.Delete("/:id", controllers.DeleteActivity) // ลบผู้ใช้
	activityRoutes.Get("/:id/enrollment-summary", controllers.GetEnrollmentSummaryByActivityID)
}
