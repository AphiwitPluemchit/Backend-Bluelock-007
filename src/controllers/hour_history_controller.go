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

// GetHourHistoryWithFilters ดึงประวัติการเปลี่ยนแปลงชั่วโมงพร้อม filters
// @Summary Get hour change history with filters
// @Description ดึงข้อมูล hour history พร้อม filter sourceType, multiple statuses, และ search title
// @Tags HourHistory
// @Accept json
// @Produce json
// @Param query query models.PaginationParams true "Pagination parameters"
// @Param filters query models.HourHistoryFilters true "Filter parameters"
// @Success 200 {object} models.HourHistoryPaginatedResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /hour-history [get]
func GetHourHistoryWithFilters(c *fiber.Ctx) error {
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

	// Get histories with filters
	histories, totalCount, err := hourhistory.GetHistoryWithFilters(
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
