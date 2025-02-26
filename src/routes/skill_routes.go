package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// SkillRoutes กำหนดเส้นทางสำหรับ Skill API
func skillRoutes(app *fiber.App) {
	skillRoutes := app.Group("/skills")
	skillRoutes.Get("/", controllers.GetSkills)         // ดึงผู้ใช้ทั้งหมด
	skillRoutes.Post("/", controllers.CreateSkill)      // สร้างผู้ใช้ใหม่
	skillRoutes.Get("/:id", controllers.GetSkillByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	skillRoutes.Put("/:id", controllers.UpdateSkill)    // อัปเดตข้อมูลผู้ใช้
	skillRoutes.Delete("/:id", controllers.DeleteSkill) // ลบผู้ใช้
}
