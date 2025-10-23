package controllers

import (
	"Backend-Bluelock-007/src/services/enrollments"
	"Backend-Bluelock-007/src/services/summary_reports"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetEnrollmentSummaryByDate ดึงข้อมูล summary จาก enrollment collection โดยตรง (แทนการใช้ Summary_Check_In_Out_Reports)
// @Summary Get enrollment summary by date (Query from enrollment directly)
// @Description ดึงข้อมูล summary ของ program ตาม date ที่ระบุ โดย query จาก enrollment collection โดยตรง รองรับ filter ตาม programItemId
// @Tags Summary Reports
// @Accept json
// @Produce json
// @Param programId path string true "Program ID"
// @Param date query string true "Date (YYYY-MM-DD)"
// @Param programItemId query string false "Program Item ID (optional - สำหรับกรณีมีหลาย programItems ในวันเดียวกัน)"
// @Success 200 {object} models.SuccessResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/summary-report/enrollment/{programId} [get]
func GetEnrollmentSummaryByDate(c *fiber.Ctx) error {
	programIDStr := c.Params("programId")
	date := c.Query("date")
	programItemIDStr := c.Query("programItemId") // optional

	if date == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Date query parameter is required (format: YYYY-MM-DD)",
		})
	}

	// แปลง string เป็น ObjectID
	programID, err := primitive.ObjectIDFromHex(programIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid program ID format",
		})
	}

	// แปลง programItemID ถ้ามี
	var programItemID *primitive.ObjectID
	if programItemIDStr != "" {
		id, err := primitive.ObjectIDFromHex(programItemIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid program item ID format",
			})
		}
		programItemID = &id
	}

	// ดึงข้อมูล summary จาก enrollment โดยตรง
	summary, err := enrollments.GetEnrollmentSummaryByDate(programID, date, programItemID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get enrollment summary: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Enrollment summary retrieved successfully",
		"data":    summary,
	})
}

// GetEnrollmentSummaryByDateV2 ดึงข้อมูล summary จาก enrollment collection โดยใช้ aggregation (ประสิทธิภาพสูงกว่า)
// @Summary Get enrollment summary by date using aggregation
// @Description ดึงข้อมูล summary ของ program ตาม date ที่ระบุ โดย query จาก enrollment collection ด้วย aggregation pipeline รองรับ filter ตาม programItemId
// @Tags Summary Reports
// @Accept json
// @Produce json
// @Param programId path string true "Program ID"
// @Param date query string true "Date (YYYY-MM-DD)"
// @Param programItemId query string false "Program Item ID (optional - สำหรับกรณีมีหลาย programItems ในวันเดียวกัน)"
// @Success 200 {object} models.SuccessResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/summary-report/enrollment-v2/{programId} [get]
func GetEnrollmentSummaryByDateV2(c *fiber.Ctx) error {
	programIDStr := c.Params("programId")
	date := c.Query("date")
	programItemIDStr := c.Query("programItemId") // optional

	if date == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Date query parameter is required (format: YYYY-MM-DD)",
		})
	}

	// แปลง string เป็น ObjectID
	programID, err := primitive.ObjectIDFromHex(programIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid program ID format",
		})
	}

	// แปลง programItemID ถ้ามี
	var programItemID *primitive.ObjectID
	if programItemIDStr != "" {
		id, err := primitive.ObjectIDFromHex(programItemIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid program item ID format",
			})
		}
		programItemID = &id
	}

	// ดึงข้อมูล summary จาก enrollment โดยตรง (V2 ใช้ aggregation)
	summary, err := enrollments.GetEnrollmentSummaryByDateV2(programID, date, programItemID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get enrollment summary: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Enrollment summary retrieved successfully (V2)",
		"data":    summary,
	})
}

// ========== API เก่าที่ใช้ Summary_Check_In_Out_Reports (เก็บไว้ backward compatibility) ==========

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
