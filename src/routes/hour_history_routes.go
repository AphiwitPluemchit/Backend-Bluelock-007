package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// hourHistoryRoutes กำหนดเส้นทางสำหรับ Hour History API
func hourHistoryRoutes(router fiber.Router) {
	hourHistoryGroup := router.Group("/hour-history")
	// hourHistoryGroup.Use(middleware.AuthJWT)

	// GET /hour-history - ดึงข้อมูล hour history พร้อม filters
	// Query params: studentId, sourceType, status (comma-separated), search, limit, page
	hourHistoryGroup.Get("/", controllers.GetHourHistoryWithFilters)
}
