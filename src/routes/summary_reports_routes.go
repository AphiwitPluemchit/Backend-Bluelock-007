package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupSummaryReportsRoutes ตั้งค่า routes สำหรับ summary reports
func SetupSummaryReportsRoutes(app fiber.Router) {
	// สร้าง group สำหรับ summary reports routes
	summaryReportsGroup := app.Group("/summary-report")
	summaryReportsGroup.Use(middleware.AuthJWT)

	// ========== NEW: API ที่ query จาก enrollment โดยตรง ==========
	// GET /api/summary-report/enrollment/:programId?date=2024-01-15 - ดึงข้อมูล summary จาก enrollment (V1)
	summaryReportsGroup.Get("/enrollment/:programId", controllers.GetEnrollmentSummaryByDate)

	// GET /api/summary-report/enrollment-v2/:programId?date=2024-01-15 - ดึงข้อมูล summary จาก enrollment (V2 - Aggregation)
	summaryReportsGroup.Get("/enrollment-v2/:programId", controllers.GetEnrollmentSummaryByDateV2)

	// ========== OLD: API เก่าที่ใช้ Summary_Check_In_Out_Reports ==========
	// GET /api/summary-reports - ดึงข้อมูล summary reports ทั้งหมด
	summaryReportsGroup.Get("/", controllers.GetAllSummaryReports)

	// GET /api/summary-reports/:programId/:date - ดึงข้อมูล summary report ของ program และ date ที่ระบุ
	summaryReportsGroup.Get("/:programId/:date", controllers.GetSummaryReportByProgramIDAndDate)

	// PUT /api/summary-reports/:programId/recalculate - คำนวณ summary reports ใหม่ทั้งหมด
	summaryReportsGroup.Put("/:programId", controllers.RecalculateSummaryReport)
}
