package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// EnrollmentRoutes กำหนดเส้นทางสำหรับ Enrollment API
func enrollmentRoutes(app *fiber.App) {
	enrollmentRoutes := app.Group("/enrollments")
	enrollmentRoutes.Post("/", controllers.CreateEnrollment)                                                                // ✅ ลงทะเบียน
	enrollmentRoutes.Get("/student/:studentId", controllers.GetEnrollmentsByStudent)                                        // ✅ ดูกิจกรรมที่ Student ลงทะเบียนไว้
	enrollmentRoutes.Delete("/:enrollmentId", controllers.DeleteEnrollment)                                                 // ✅ ยกเลิกลงทะเบียน
	enrollmentRoutes.Get("/activity/:activityId", controllers.GetStudentsByActivity)                                        // ✅ Admin ดูนักศึกษาที่ลงทะเบียน
	enrollmentRoutes.Get("/student/:studentId/activityItem/:activityItemId", controllers.GetEnrollmentByStudentAndActivity) // ✅ ดูกิจกรรมที่ลงทะเบียน (1 ตัว)
	enrollmentRoutes.Get("/student/:studentId/activity/:activityId", controllers.CheckEnrollmentByStudentAndActivity)       // ✅ ตรวจสอบกิจกรรมที่ลงทะเบียน

}
