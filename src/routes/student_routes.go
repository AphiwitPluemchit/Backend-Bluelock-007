package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// StudentRoutes กำหนดเส้นทางสำหรับ Student API
func studentRoutes(app *fiber.App) {
	studentRoutes := app.Group("/students")
	studentRoutes.Get("/", controllers.GetStudents)           // ดึงผู้ใช้ทั้งหมด
	studentRoutes.Post("/", controllers.CreateStudent)        // สร้างผู้ใช้ใหม่
	studentRoutes.Get("/:code", controllers.GetStudentByCode) // ดึงข้อมูลผู้ใช้ตาม ID
	studentRoutes.Put("/:id", controllers.UpdateStudent)      // อัปเดตข้อมูลผู้ใช้
	studentRoutes.Delete("/:id", controllers.DeleteStudent)   // ลบผู้ใช้
}
