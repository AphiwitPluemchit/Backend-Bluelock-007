package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// SuggestionRoutes กำหนดเส้นทางสำหรับ Suggestion API
func suggestionRoutes(router fiber.Router) {
	suggestionRoutes := router.Group("/suggestions")
	suggestionRoutes.Use(middleware.AuthJWT)
	suggestionRoutes.Get("/", controllers.GetSuggestions)         // ดึงผู้ใช้ทั้งหมด
	suggestionRoutes.Post("/", controllers.CreateSuggestion)      // สร้างผู้ใช้ใหม่
	suggestionRoutes.Get("/:id", controllers.GetSuggestionByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	suggestionRoutes.Put("/:id", controllers.UpdateSuggestion)    // อัปเดตข้อมูลผู้ใช้
	suggestionRoutes.Delete("/:id", controllers.DeleteSuggestion) // ลบผู้ใช้
}
