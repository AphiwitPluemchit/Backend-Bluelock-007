package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// ProgramRoutes กำหนดเส้นทางสำหรับ Program API
func programRoutes(router fiber.Router) {
	programRoutes := router.Group("/programs")
	programRoutes.Use(middleware.AuthJWT)
	programRoutes.Get("/", controllers.GetAllPrograms) // ดึงผู้ใช้ทั้งหมด
	programRoutes.Post("/", controllers.CreateProgram) // สร้างผู้ใช้ใหม่
	programRoutes.Post(":id/image", controllers.UploadProgramImage)
	programRoutes.Delete(":id/image", controllers.DeleteProgramImage)
	programRoutes.Get("/:id", controllers.GetProgramByID)   // ดึงข้อมูลผู้ใช้ตาม ID
	programRoutes.Put("/:id", controllers.UpdateProgram)    // อัปเดตข้อมูลผู้ใช้
	programRoutes.Delete("/:id", controllers.DeleteProgram) // ลบผู้ใช้
	programRoutes.Get("/:id/enrollment-summary", controllers.GetEnrollmentSummaryByProgramID)

	programRoutes.Get("/calendar/:month/:year", controllers.GetAllProgramCalendar)
}
