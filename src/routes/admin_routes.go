package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// AdminRoutes กำหนดเส้นทางสำหรับ Admin API
func adminRoutes(app *fiber.App) {
	adminRoutes := app.Group("/admins")
	adminRoutes.Get("/", controllers.GetAdmins)         // ดึงผู้ใช้ทั้งหมด
	adminRoutes.Post("/", controllers.CreateAdmin)      // สร้างผู้ใช้ใหม่
	adminRoutes.Get("/:id", controllers.GetAdminByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	adminRoutes.Put("/:id", controllers.UpdateAdmin)    // อัปเดตข้อมูลผู้ใช้
	adminRoutes.Delete("/:id", controllers.DeleteAdmin) // ลบผู้ใช้
}
