package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// FoodVoteRoutes กำหนดเส้นทางสำหรับ FoodVote API
func foodVoteRoutes(app *fiber.App) {
	foodVoteRoutes := app.Group("/foodVotes")
	foodVoteRoutes.Get("/", controllers.GetActivitys)         // ดึงผู้ใช้ทั้งหมด
	foodVoteRoutes.Post("/", controllers.CreateActivity)      // สร้างผู้ใช้ใหม่
	foodVoteRoutes.Get("/:id", controllers.GetActivityByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	foodVoteRoutes.Put("/:id", controllers.UpdateActivity)    // อัปเดตข้อมูลผู้ใช้
	foodVoteRoutes.Delete("/:id", controllers.DeleteActivity) // ลบผู้ใช้
}
