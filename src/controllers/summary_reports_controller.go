package controllers

import (
	"Backend-Bluelock-007/src/services/summary_reports"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetSummaryReportByProgramID ดึงข้อมูล summary report ของ program
func GetSummaryReportByProgramID(c *fiber.Ctx) error {
	programIDStr := c.Params("programId")

	// แปลง string เป็น ObjectID
	programID, err := primitive.ObjectIDFromHex(programIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid program ID format",
		})
	}

	// ดึงข้อมูล summary reports ทั้งหมดของ program
	summaries, err := summary_reports.GetSummaryReport(programID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Summary reports not found for this program",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Summary reports retrieved successfully",
		"data":    summaries,
	})
}

// GetSummaryReportByProgramIDAndDate ดึงข้อมูล summary report ของ program และ date ที่ระบุ
func GetSummaryReportByProgramIDAndDate(c *fiber.Ctx) error {
	programIDStr := c.Params("programId")
	date := c.Params("date")

	// แปลง string เป็น ObjectID
	programID, err := primitive.ObjectIDFromHex(programIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid program ID format",
		})
	}

	// ดึงข้อมูล summary report สำหรับ date ที่ระบุ
	summary, err := summary_reports.GetSummaryReportByDate(programID, date)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Summary report not found for this program and date",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Summary report retrieved successfully",
		"data":    summary,
	})
}

// GetAllSummaryReports ดึงข้อมูล summary reports ทั้งหมด
func GetAllSummaryReports(c *fiber.Ctx) error {
	// ดึงข้อมูล summary reports ทั้งหมด
	summaries, err := summary_reports.GetAllSummaryReports()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve summary reports",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Summary reports retrieved successfully",
		"data":    summaries,
	})
}

// RecalculateSummaryReport คำนวณ summary report ใหม่สำหรับ program
func RecalculateSummaryReport(c *fiber.Ctx) error {
	programIDStr := c.Params("programId")

	// แปลง string เป็น ObjectID
	programID, err := primitive.ObjectIDFromHex(programIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid program ID format",
		})
	}

	// ดึงข้อมูล summary reports ทั้งหมดของ program ก่อน
	summaries, err := summary_reports.GetSummaryReport(programID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Summary reports not found for this program",
		})
	}

	// คำนวณ NotParticipating ใหม่สำหรับแต่ละ date
	for _, summary := range summaries {
		err = summary_reports.RecalculateNotParticipating(programID, summary.Date)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to recalculate summary report for date: " + summary.Date,
			})
		}
	}

	// ดึงข้อมูล summary reports ที่อัปเดตแล้ว
	summaries, err = summary_reports.GetSummaryReport(programID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Summary reports not found after recalculation",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Summary reports recalculated successfully",
		"data":    summaries,
	})
}
