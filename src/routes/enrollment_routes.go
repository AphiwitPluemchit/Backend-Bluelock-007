package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// EnrollmentRoutes กำหนดเส้นทางสำหรับ Enrollment API
func enrollmentRoutes(app *fiber.App) {
	enrollmentRoutes := app.Group("/enrollments")
	enrollmentRoutes.Post("/", controllers.CreateEnrollment)
	enrollmentRoutes.Get("/student/:studentId", controllers.GetEnrollmentsByStudent)
	enrollmentRoutes.Delete("/student/:studentId/activity/:activityItemId", controllers.DeleteEnrollment)
	enrollmentRoutes.Get("/activity/:activityItemId", controllers.GetStudentsByActivity)
	enrollmentRoutes.Get("/student/:studentId/activity/:activityItemId", controllers.GetEnrollmentByStudentAndActivity)
}
