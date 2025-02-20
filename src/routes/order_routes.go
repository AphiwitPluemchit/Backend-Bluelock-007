package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// InitOrderRoutes กำหนด routes สำหรับ orders
func OrderRoutes(app *fiber.App) {
	orderRoutes := app.Group("/orders")
	orderRoutes.Post("/", controllers.CreateOrder) // Create new order

	orderRoutes.Get("/", controllers.GetOrders) // Get all orders

}
