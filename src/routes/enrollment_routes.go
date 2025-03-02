package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// EnrollmentRoutes กำหนดเส้นทางสำหรับ Enrollment API
func enrollmentRoutes(app *fiber.App) {
	enrollmentRoutes := app.Group("/enrollments")
	enrollmentRoutes.Get("/", controllers.GetAllEnrollments)                                                            // ดึงผู้ใช้ทั้งหมด
	enrollmentRoutes.Post("/", controllers.CreateEnrollment)                                                            // สร้างผู้ใช้ใหม่
	enrollmentRoutes.Get("/student/:studentId", controllers.GetEnrollmentsByStudent)                                    // ดูกิจกรรมที่ลงทะเบียนทั้งหมด
	enrollmentRoutes.Get("/student/:studentId/activity/:activityItemId", controllers.GetEnrollmentByStudentAndActivity) // ดูเฉพาะกิจกรรมที่เลือก
	enrollmentRoutes.Get("/:id", controllers.GetEnrollmentByID)                                                         // ดึงข้อมูลผู้ใช้ตาม ID
	enrollmentRoutes.Put("/:id", controllers.UpdateEnrollment)                                                          // อัปเดตข้อมูลผู้ใช้
	enrollmentRoutes.Delete("/:id", controllers.DeleteEnrollment)                                                       // ลบผู้ใช้
}
