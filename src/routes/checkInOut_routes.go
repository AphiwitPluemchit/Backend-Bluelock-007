package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// CheckInOutRoutes กำหนดเส้นทางสำหรับ CheckInOut API
func checkInOutRoutes(app *fiber.App) {
	checkInOutRoutes := app.Group("/checkInOuts")
	checkInOutRoutes.Use(middleware.AuthJWT)
	checkInOutRoutes.Post("/generate-link", controllers.GenerateLink)
	checkInOutRoutes.Post("/checkin/:uuid", controllers.Checkin) // ดึงผู้ใช้ทั้งหมด

}
