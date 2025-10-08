package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// EnrollmentRoutes กำหนดเส้นทางสำหรับ Enrollment API
func enrollmentRoutes(router fiber.Router) {
	enrollmentRoutes := router.Group("/enrollments")
	enrollmentRoutes.Post("/", controllers.RegisterStudent) // ✅ ลงทะเบียน
	// enrollmentRoutes.Post("/many", controllers.CreateBulkEnrollment)       // ✅ ลงทะเบียนหลายคน                                              // ✅ ลงทะเบียนหลายคน
	enrollmentRoutes.Get("/student/:studentId", controllers.GetEnrollmentsByStudent) // ✅ ดูกิจกรรมที่ Student ลงทะเบียนไว้
	enrollmentRoutes.Get("/:enrollmentId", controllers.GetEnrollmentById)
	enrollmentRoutes.Patch("/:enrollmentId/checkinout", controllers.UpdateEnrollmentCheckinout)
	enrollmentRoutes.Delete("/:enrollmentId", controllers.UnregisterStudent) // ✅ ยกเลิกลงทะเบียน
	// enrollmentRoutes.Get("/program/:programId", controllers.GetStudentsByProgram)                                        // ✅ Admin ดูนักศึกษาที่ลงทะเบียน
	enrollmentRoutes.Get("/student/:studentId/program/:programId/check", controllers.CheckEnrollmentByStudentAndProgram) // ✅ ตรวจสอบว่านักศึกษาลงทะเบียนในกิจกรรมหรือไม่

	// ดูนิสิตที่ลงทะเบียน
	enrollmentRoutes.Get("/programItems/:id/enrollments", controllers.GetEnrollmentByProgramItemID) //programItems enrollments
	enrollmentRoutes.Get("/:id/enrollments", controllers.GetEnrollmentsByProgramID)                 //program enrollments

	// enrollmentRoutes.Get("/student/:studentId/program/:programId", controllers.GetStudentEnrollmentInProgram)            // ✅ ดึงข้อมูล Enrollment ของ Student ใน Program
	// enrollmentRoutes.Get("/history/student/:studentId", controllers.GetRegistrationHistoryStatus)          // ✅ ประวัติการลงทะเบียน แบ่งสถานะจาก Hour_Change_Histories
	// enrollmentRoutes.Get("/history-status/student/:studentId", controllers.GetEnrollmentsHistoryByStudent) // ✅ ประวัติการอบรมของ Student (กิจกรรมทั้งหมดที่เคยลง)
}
