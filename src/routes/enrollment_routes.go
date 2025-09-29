package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// EnrollmentRoutes กำหนดเส้นทางสำหรับ Enrollment API
func enrollmentRoutes(router fiber.Router) {
	enrollmentRoutes := router.Group("/enrollments")
	enrollmentRoutes.Post("/", controllers.CreateEnrollment) // ✅ ลงทะเบียน
	// enrollmentRoutes.Post("/many", controllers.CreateBulkEnrollment)                                                     // ✅ ลงทะเบียนหลายคน
	enrollmentRoutes.Get("/student/:studentId", controllers.GetEnrollmentsByStudent) // ✅ ดูกิจกรรมที่ Student ลงทะเบียนไว้
	enrollmentRoutes.Delete("/:enrollmentId", controllers.DeleteEnrollment)          // ✅ ยกเลิกลงทะเบียน
	// enrollmentRoutes.Get("/program/:programId", controllers.GetStudentsByProgram)                                        // ✅ Admin ดูนักศึกษาที่ลงทะเบียน
	enrollmentRoutes.Get("/student/:studentId/program/:programId/check", controllers.CheckEnrollmentByStudentAndProgram) // ✅ ตรวจสอบว่านักศึกษาลงทะเบียนในกิจกรรมหรือไม่

	// enrollmentRoutes.Get("/student/:studentId/program/:programId", controllers.GetStudentEnrollmentInProgram)            // ✅ ดึงข้อมูล Enrollment ของ Student ใน Program
	// enrollmentRoutes.Get("/history/student/:studentId", controllers.GetRegistrationHistoryStatus)          // ✅ ประวัติการลงทะเบียน แบ่งสถานะจาก Hour_Change_Histories
	// enrollmentRoutes.Get("/history-status/student/:studentId", controllers.GetEnrollmentsHistoryByStudent) // ✅ ประวัติการอบรมของ Student (กิจกรรมทั้งหมดที่เคยลง)
}
