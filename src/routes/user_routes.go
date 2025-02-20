package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// UserRoutes กำหนดเส้นทางสำหรับ User API
func UserRoutes(app *fiber.App) {
	userRoutes := app.Group("/users")
	userRoutes.Get("/", controllers.GetUsers)         // ดึงผู้ใช้ทั้งหมด
	userRoutes.Post("/", controllers.CreateUser)      // สร้างผู้ใช้ใหม่
	userRoutes.Get("/:id", controllers.GetUserByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	userRoutes.Put("/:id", controllers.UpdateUser)    // อัปเดตข้อมูลผู้ใช้
	userRoutes.Delete("/:id", controllers.DeleteUser) // ลบผู้ใช้
}
