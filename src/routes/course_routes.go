package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// CourseRoutes จัดการเส้นทางสำหรับ Course API
func courseRoutes(router fiber.Router) {
	courseRoutes := router.Group("/courses")
	courseRoutes.Get("/", controllers.GetAllCourses)
	courseRoutes.Post("/", controllers.CreateCourse)
	courseRoutes.Get("/:id", controllers.GetCourseByID)
	courseRoutes.Put("/:id", controllers.UpdateCourse)
	courseRoutes.Delete("/:id", controllers.DeleteCourse)
}
