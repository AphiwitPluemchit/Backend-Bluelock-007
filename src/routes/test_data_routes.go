package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// TestDataRoutes กำหนด routes สำหรับสร้างข้อมูลทดสอบ
func TestDataRoutes(app fiber.Router) {
	testGroup := app.Group("/api/test")

	// สร้างข้อมูลทดสอบ enrollment (ไม่รวม check-in/out)
	testGroup.Post("/enrollment", controllers.CreateTestEnrollment)

	// อัปเดต check-in/out records
	testGroup.Put("/checkinout", controllers.UpdateCheckInOutRecords)

	// ลบข้อมูลทดสอบ enrollment
	testGroup.Delete("/enrollment/:enrollmentId", controllers.DeleteTestEnrollment)
}
