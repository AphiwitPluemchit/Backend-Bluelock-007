package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// hourHistoryRoutes กำหนดเส้นทางสำหรับ Hour History API
func hourHistoryRoutes(router fiber.Router) {
	hourHistoryGroup := router.Group("/hour-history")
	// hourHistoryGroup.Use(middleware.AuthJWT)

	// GET /hour-history/details - ดึงข้อมูล hour history พร้อม ProgramItem และ Certificate details
	// Query params: studentId, sourceType, status (comma-separated), search, limit, page
	hourHistoryGroup.Get("/details", controllers.GetHourHistoryWithDetails)

	// GET /hour-history/student-hours-summary - ดึงชั่วโมงรวมของนิสิตจาก hour history
	// Query params: studentId (required)
	hourHistoryGroup.Get("/student-hours-summary", controllers.GetStudentHoursSummary)

	// POST /hour-history/direct - สร้างการเปลี่ยนแปลงชั่วโมงโดยตรงโดย Admin
	// Body: CreateDirectHourChangeRequest
	hourHistoryGroup.Post("/direct", controllers.CreateDirectHourChange)
}
