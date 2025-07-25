package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// FoodRoutes กำหนดเส้นทางสำหรับ Food API
func foodRoutes(router fiber.Router) {
	foodRoutes := router.Group("/foods")
	foodRoutes.Use(middleware.AuthJWT)
	foodRoutes.Get("/", controllers.GetFoods)
	foodRoutes.Post("/", controllers.CreateFood)
	foodRoutes.Get("/:id", controllers.GetFoodByID)
	foodRoutes.Put("/:id", controllers.UpdateFood)
	foodRoutes.Delete("/:id", controllers.DeleteFood)
}
