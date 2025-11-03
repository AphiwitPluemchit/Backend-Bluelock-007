package controllers

import (
	"Backend-Bluelock-007/src/models"
	hourhistory "Backend-Bluelock-007/src/services/hour-history"
	"Backend-Bluelock-007/src/utils"
	"math"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetHourHistoryWithDetails ดึงประวัติการเปลี่ยนแปลงชั่วโมงพร้อม details จาก ProgramItem และ Certificate
// @Summary Get hour change history with program item and certificate details
// @Description ดึงข้อมูล hour history พร้อม populate ข้อมูล Program, ProgramItem และ Certificate
// @Tags HourHistory
// @Accept json
// @Produce json
// @Param query query models.PaginationParams true "Pagination parameters"
// @Param filters query models.HourHistoryFilters true "Filter parameters"
// @Success 200 {object} models.HourHistoryPaginatedResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /hour-history/details [get]
func GetHourHistoryWithDetails(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse pagination parameters
	params := models.DefaultPagination()
	if err := c.QueryParser(&params); err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid query parameters")
	}

	// Parse filter parameters
	var filters models.HourHistoryFilters
	if err := c.QueryParser(&filters); err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid filter parameters")
	}

	// Parse studentID (optional)
	var studentID *primitive.ObjectID
	if filters.StudentID != "" {
		objID, err := primitive.ObjectIDFromHex(filters.StudentID)
		if err != nil {
			return utils.HandleError(c, fiber.StatusBadRequest, "Invalid studentId format")
		}
		studentID = &objID
	}

	// Parse statuses (support multiple statuses separated by comma)
	var statuses []string
	if filters.Status != "" {
		statuses = strings.Split(filters.Status, ",")
		// Trim spaces
		for i, status := range statuses {
			statuses[i] = strings.TrimSpace(status)
		}
	}

	// Calculate skip for pagination
	skip := (params.Page - 1) * params.Limit

	// Get histories with details (populated)
	histories, totalCount, err := hourhistory.GetHistoryWithDetailsAndFilters(
		ctx,
		studentID,
		filters.SourceType,
		statuses,
		filters.Search,
		params.Limit,
		skip,
	)

	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(totalCount) / float64(params.Limit)))

	// Build response
	response := models.HourHistoryPaginatedResponse{
		Data: histories,
		Meta: models.PaginationMeta{
			Page:       params.Page,
			Limit:      params.Limit,
			Total:      totalCount,
			TotalPages: totalPages,
		},
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetStudentHoursSummary - ดึงชั่วโมงรวมของนิสิตจาก hour history
func GetStudentHoursSummary(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse studentID from query parameter
	studentIDStr := c.Query("studentId")
	if studentIDStr == "" {
		return utils.HandleError(c, fiber.StatusBadRequest, "studentId is required")
	}

	studentID, err := primitive.ObjectIDFromHex(studentIDStr)
	if err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid studentId format")
	}

	// Get hours summary from hour history
	summary, err := hourhistory.GetStudentHoursSummary(ctx, studentID)
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(summary)
}

// CreateDirectHourChange สร้างการเปลี่ยนแปลงชั่วโมงโดยตรงโดย Admin
// @Summary Create direct hour change by admin
// @Description สร้างการเปลี่ยนแปลงชั่วโมงโดยตรงโดย Admin โดยไม่ต้องผ่าน program หรือ certificate
// @Tags HourHistory
// @Accept json
// @Produce json
// @Param body body models.CreateDirectHourChangeRequest true "Direct hour change data"
// @Success 201 {object} models.HourChangeHistory
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /hour-history/direct [post]
func CreateDirectHourChange(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse request body
	var req models.CreateDirectHourChangeRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if req.StudentID == "" {
		return utils.HandleError(c, fiber.StatusBadRequest, "studentId is required")
	}
	if req.Title == "" {
		return utils.HandleError(c, fiber.StatusBadRequest, "title is required")
	}
	if req.SourceType == "" {
		return utils.HandleError(c, fiber.StatusBadRequest, "sourceType is required")
	}
	if req.SkillType == "" {
		return utils.HandleError(c, fiber.StatusBadRequest, "skillType is required")
	}
	if req.HourChange == 0 {
		return utils.HandleError(c, fiber.StatusBadRequest, "hourChange cannot be zero")
	}

	// Validate enums
	if req.SourceType != "program" && req.SourceType != "certificate" {
		return utils.HandleError(c, fiber.StatusBadRequest, "sourceType must be 'program' or 'certificate'")
	}
	if req.SkillType != "soft" && req.SkillType != "hard" {
		return utils.HandleError(c, fiber.StatusBadRequest, "skillType must be 'soft' or 'hard'")
	}

	// Parse studentID
	studentID, err := primitive.ObjectIDFromHex(req.StudentID)
	if err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid studentId format")
	}

	// Create direct hour change
	history, err := hourhistory.CreateHourChangeHistory(
		ctx,
		studentID,
		req.SourceType,
		nil, // ไม่มี sourceID สำหรับ manual entry
		req.SkillType,
		models.HCStatusManual,
		req.HourChange,
		req.Title,
		req.Remark,
		nil, // ไม่มี enrollmentID
		nil, // ไม่มี programItemID
	)

	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(history)
}
