package routes

import (
	"Backend-Bluelock-007/src/controllers"

	"github.com/gofiber/fiber/v2"
)

// SetupSummaryReportsRoutes ตั้งค่า routes สำหรับ summary reports
func SetupSummaryReportsRoutes(app fiber.Router) {
	// สร้าง group สำหรับ summary reports routes
	summaryReportsGroup := app.Group("/summary-report")

	// GET /api/summary-reports - ดึงข้อมูล summary reports ทั้งหมด
	summaryReportsGroup.Get("/", controllers.GetAllSummaryReports)

	// GET /api/summary-reports/:programId - ดึงข้อมูล summary reports ทั้งหมดของ program
	summaryReportsGroup.Get("/:programId", controllers.GetSummaryReportByProgramID)

	// GET /api/summary-reports/:programId/:date - ดึงข้อมูล summary report ของ program และ date ที่ระบุ
	summaryReportsGroup.Get("/:programId/:date", controllers.GetSummaryReportByProgramIDAndDate)

	// PUT /api/summary-reports/:programId/recalculate - คำนวณ summary reports ใหม่ทั้งหมด
	summaryReportsGroup.Put("/:programId", controllers.RecalculateSummaryReport)
}
