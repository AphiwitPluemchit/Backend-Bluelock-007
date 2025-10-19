package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// StudentRoutes กำหนดเส้นทางสำหรับ Student API
func studentRoutes(router fiber.Router) {
	studentGroup := router.Group("/students")
	studentGroup.Use(middleware.AuthJWT)
	studentGroup.Get("/", controllers.GetStudents)                                   // ดึงผู้ใช้ทั้งหมด
	studentGroup.Post("/", controllers.CreateStudent)                                // สร้างผู้ใช้ใหม่
	studentGroup.Get("/:code", controllers.GetStudentByCode)                         // ดึงข้อมูลผู้ใช้ตาม ID
	studentGroup.Put("/:id", controllers.UpdateStudent)                              // อัปเดตข้อมูลผู้ใช้
	studentGroup.Delete("/:id", controllers.DeleteStudent)                           // ลบผู้ใช้
	studentGroup.Get("/report/sammary-all", controllers.GetSammaryAll)               // ดึงข้อมูลสรุปทั้งหมด
	studentGroup.Get("/sammary/:code", controllers.GetSammaryByCode)                 // ดึงข้อมูลสรุปตามรหัส
	studentGroup.Post("/update-status-by-ids", controllers.UpdateStudentStatusByIDs) // เพิ่ม route ใหม่
	// studentGroup.Put("/update-status/:id", controllers.UpdateStudentStatus)          // อัปเดตสถานะนักเรียน
}
