package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// CheckInOutRoutes กำหนดเส้นทางสำหรับ CheckInOut API
func checkInOutRoutes(router fiber.Router) {
	checkInOutRoutes := router.Group("/checkInOuts")
	// checkInOutRoutes.Use(middleware.AuthJWT)
	checkInOutRoutes.Post("/generate-link", controllers.GenerateLink)
	checkInOutRoutes.Post("/checkin/:uuid", controllers.Checkin)   // ดึงผู้ใช้ทั้งหมด
	checkInOutRoutes.Post("/checkout/:uuid", controllers.Checkout) // ดึงผู้ใช้ทั้งหมด
	checkInOutRoutes.Get("/status", controllers.GetCheckinStatus)
}
