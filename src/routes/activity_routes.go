package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// ActivityRoutes กำหนดเส้นทางสำหรับ Activity API
func activityRoutes(router fiber.Router) {
	activityRoutes := router.Group("/activitys")
	activityRoutes.Use(middleware.AuthJWT)
	activityRoutes.Get("/", controllers.GetAllActivities) // ดึงผู้ใช้ทั้งหมด
	activityRoutes.Post("/", controllers.CreateActivity)  // สร้างผู้ใช้ใหม่
	activityRoutes.Post(":id/image", controllers.UploadActivityImage)
	activityRoutes.Delete(":id/image", controllers.DeleteActivityImage)
	activityRoutes.Get("/:id", controllers.GetActivityByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	activityRoutes.Put("/:id", controllers.UpdateActivity)    // อัปเดตข้อมูลผู้ใช้
	activityRoutes.Delete("/:id", controllers.DeleteActivity) // ลบผู้ใช้
	activityRoutes.Get("/:id/enrollment-summary", controllers.GetEnrollmentSummaryByActivityID)
	activityRoutes.Get("/:id/enrollments", controllers.GetEnrollmentByActivityItemID)
	activityRoutes.Get("/calendar/:month/:year", controllers.GetAllActivityCalendar)
}
