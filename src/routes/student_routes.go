package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"log"

	"github.com/gofiber/fiber/v2"
)

// StudentRoutes กำหนดเส้นทางสำหรับ Student API
func studentRoutes(router fiber.Router) {
	studentRoutes := router.Group("/students")
	studentRoutes.Get("/", controllers.GetStudents)                       // ดึงผู้ใช้ทั้งหมด
	studentRoutes.Post("/", controllers.CreateStudent)                    // สร้างผู้ใช้ใหม่
	studentRoutes.Get("/:code", controllers.GetStudentByCode)             // ดึงข้อมูลผู้ใช้ตาม ID
	studentRoutes.Put("/:id", controllers.UpdateStudent)                  // อัปเดตข้อมูลผู้ใช้
	studentRoutes.Delete("/:id", controllers.DeleteStudent)               // ลบผู้ใช้
	studentRoutes.Post("/update-status", controllers.UpdateStudentStatus) // สร้างผู้ใช้ใหม่
	studentRoutes.Get("/report/sammary-all", controllers.GetSammaryAll)

	studentRoutes.Get("/sammary/:code", controllers.GetSammaryByCode)
	log.Println("Register route /students/sammary-all")
}
