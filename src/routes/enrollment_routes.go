package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// EnrollmentRoutes กำหนดเส้นทางสำหรับ Enrollment API
func enrollmentRoutes(router fiber.Router) {
	enrollmentRoutes := router.Group("/enrollments")
	enrollmentRoutes.Use(middleware.AuthJWT)
	enrollmentRoutes.Post("/", controllers.CreateEnrollment)                                                                // ✅ ลงทะเบียน
	enrollmentRoutes.Get("/student/:studentId", controllers.GetEnrollmentsByStudent)                                        // ✅ ดูกิจกรรมที่ Student ลงทะเบียนไว้
	enrollmentRoutes.Delete("/:enrollmentId", controllers.DeleteEnrollment)                                                 // ✅ ยกเลิกลงทะเบียน
	enrollmentRoutes.Get("/activity/:activityId", controllers.GetStudentsByActivity)                                        // ✅ Admin ดูนักศึกษาที่ลงทะเบียน
	enrollmentRoutes.Get("/student/:studentId/activity/:activityId/check", controllers.CheckEnrollmentByStudentAndActivity) // ✅ ตรวจสอบว่านักศึกษาลงทะเบียนในกิจกรรมหรือไม่
	enrollmentRoutes.Get("/student/:studentId/activity/:activityId", controllers.GetStudentEnrollmentInActivity)            // ✅ ดึงข้อมูล Enrollment ของ Student ใน Activity
}
