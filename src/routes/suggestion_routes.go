package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// SuggestionRoutes กำหนดเส้นทางสำหรับ Suggestion API
func suggestionRoutes(app *fiber.App) {
	suggestionRoutes := app.Group("/suggestions")
	suggestionRoutes.Get("/", controllers.GetActivitys)         // ดึงผู้ใช้ทั้งหมด
	suggestionRoutes.Post("/", controllers.CreateActivity)      // สร้างผู้ใช้ใหม่
	suggestionRoutes.Get("/:id", controllers.GetActivityByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	suggestionRoutes.Put("/:id", controllers.UpdateActivity)    // อัปเดตข้อมูลผู้ใช้
	suggestionRoutes.Delete("/:id", controllers.DeleteActivity) // ลบผู้ใช้
}
