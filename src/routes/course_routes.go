package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// CourseRoutes จัดการเส้นทางสำหรับ Course API
func courseRoutes(router fiber.Router) {
	courseRoutes := router.Group("/courses")
	// courseRoutes.Use(middleware.AuthJWT)
	courseRoutes.Get("/", controllers.GetAllCourses)
	courseRoutes.Post("/", controllers.CreateCourse)
	courseRoutes.Get("/:id", controllers.GetCourseByID)
	courseRoutes.Put("/:id", controllers.UpdateCourse)
	courseRoutes.Delete("/:id", controllers.DeleteCourse)

	// Image upload/delete endpoints
	courseRoutes.Post("/:id/image", controllers.UploadCourseImage)
	courseRoutes.Delete("/:id/image", controllers.DeleteCourseImage)
}
