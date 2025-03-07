package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// FoodVoteRoutes กำหนดเส้นทางสำหรับ FoodVote API
func foodVoteRoutes(app *fiber.App) {
	foodVoteRoutes := app.Group("/foodVotes")
	// foodVoteRoutes.Get("/", controllers.GetFoodVotes)         // ดึงผู้ใช้ทั้งหมด
	foodVoteRoutes.Post("/", controllers.CreateFoodVote)      // สร้างผู้ใช้ใหม่
	foodVoteRoutes.Get("/:id", controllers.GetFoodVoteByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	foodVoteRoutes.Put("/:id", controllers.UpdateFoodVote)    // อัปเดตข้อมูลผู้ใช้
	foodVoteRoutes.Delete("/:id", controllers.DeleteFoodVote) // ลบผู้ใช้
}
