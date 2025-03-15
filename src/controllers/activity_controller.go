package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"
	"Backend-Bluelock-007/src/utils"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateActivity godoc
// @Summary      Create a new activity
// @Description  Create a new activity
// @Tags         activitys
// @Accept       json
// @Produce      json
// @Param        body body models.Activity true "Activity and ActivityItems"
// @Success      201  {object}  models.Activity
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /activitys [post]
// CreateActivity - สร้างกิจกรรมใหม่
func CreateActivity(c *fiber.Ctx) error {
	var request models.Activity

	// แปลง JSON เป็น struct
	if err := c.BodyParser(&request); err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid input: "+err.Error())
	}

	// บันทึก Activity + Items
	activity, err := services.CreateActivity(&request)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Activity and ActivityItems created successfully",
		"data":    activity,
	})
}

// GetAllActivities godoc
// @Summary      Get all activities with pagination, search, and sorting
// @Description  Get all activities with pagination, search, and sorting
// @Tags         activitys
// @Produce      json
// @Param        page   query  int  false  "Page number" default(1)
// @Param        limit  query  int  false  "Number of items per page" default(10)
// @Param        search query  string  false  "Search term"
// @Param        sortBy query  string  false  "Field to sort by" default(name)
// @Param        order  query  string  false  "Sort order (asc or desc)" default(asc)
// @Param        skill          query  string  false  "Filter by skill"
// @Param        activityState  query  string  false  "Filter by activityState"
// @Param        major          query  string  false  "Filter by major"
// @Param        studentYear    query  string  false  "Filter by studentYear"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  models.ErrorResponse
// @Router       /activitys [get]
func GetAllActivities(c *fiber.Ctx) error {
	// ใช้ DTO Default แล้วอัปเดตค่าจาก Query Parameter
	params := models.DefaultPagination()

	// อ่านค่า Query Parameter และแปลงเป็น int
	params.Page, _ = strconv.Atoi(c.Query("page", strconv.Itoa(params.Page)))
	params.Limit, _ = strconv.Atoi(c.Query("limit", strconv.Itoa(params.Limit)))
	params.Search = c.Query("search", params.Search)
	params.SortBy = c.Query("sortBy", params.SortBy)
	params.Order = c.Query("order", params.Order)

	// ดึงค่าจาก Query Parameters และแปลงเป็น array
	skills := c.Query("skill")                 // เช่น skill=soft,hard
	activityStates := c.Query("activityState") // เช่น activityState=open,planning
	majors := c.Query("major")                 // เช่น major=CS,SE
	studentYears := c.Query("studentYear")     // เช่น studentYear=1,2,3

	// Convert comma-separated values into arrays
	skillFilter := strings.Split(skills, ",")
	stateFilter := strings.Split(activityStates, ",")
	majorFilter := strings.Split(majors, ",")
	// Convert studentYear to int array
	yearFilter := make([]int, 0)
	for _, yearStr := range strings.Split(studentYears, ",") {
		year, err := strconv.Atoi(yearStr)
		if err == nil {
			yearFilter = append(yearFilter, year)
		}
	}

	// ดึงข้อมูลจาก Service
	activities, total, totalPages, err := services.GetAllActivities(params, skillFilter, stateFilter, majorFilter, yearFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch activities",
		})
	}

	// ส่ง Response กลับไป
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": activities,
		"meta": fiber.Map{
			"page":       params.Page,
			"limit":      params.Limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

// GetActivityByID - ดึงข้อมูลกิจกรรมตาม ID พร้อม ActivityItems
// GetActivityByID - godoc
// @Summary      Get an activity by ID
// @Description  Get an activity by ID
// @Tags         activitys
// @Produce      json
// @Param        id   path  string  true  "Activity ID"
// @Success      200  {object}  models.Activity
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /activitys/{id} [get]
func GetActivityByID(c *fiber.Ctx) error {
	id := c.Params("id")
	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// ดึงข้อมูล Activity พร้อม ActivityItems
	activity, err := services.GetActivityByID(activityID.Hex())
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Activity not found"})
	}

	// ส่งข้อมูลกลับรวมทั้ง ActivityItems
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": activity,
	})
}

// GetEnrollmentSummaryByActivityID - ดึงข้อมูลสรุปการลงทะเบียน
// GetEnrollmentSummaryByActivityID - godoc
// @Summary      Get enrollment summary by activity ID
// @Description  Get enrollment summary by activity ID
// @Tags         activitys
// @Produce      json
// @Param        id   path  string  true  "Activity ID"
// @Success      200  {object} 	models.EnrollmentSummary
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /activitys/{id}/enrollment-summary [get]
func GetEnrollmentSummaryByActivityID(c *fiber.Ctx) error {
	id := c.Params("id")
	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// ดึงข้อมูลสรุปการลงทะเบียน
	enrollmentSummary, err := services.GetActivityEnrollSummary(activityID.Hex())
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Activity not found",
			"message": err.Error(),
		})
	}

	// ส่งข้อมูลกลับ
	return c.Status(fiber.StatusOK).JSON(enrollmentSummary)
}

// GetEnrollmentByActivityID - ดึงข้อมูลการลงทะเบียนตาม ID กิจกรรม
// GetEnrollmentByActivityID - godoc
// @Summary      Get enrollments by activity ID
// @Description  Get enrollments by activity ID
// @Tags         activitys
// @Produce      json
// @Param        id   path  string  true  "Activity ID"
// @Param        page   query  int  false  "Page number"
// @Param        limit   query  int  false  "Items per page"
// @Param        search   query  string  false  "Search query"
// @Param        sortBy   query  string  false  "Sort by field"
// @Param        order    query  string  false  "Sort order"
// @Param        majors   query  string  false  "Filter by majors"
// @Param        status   query  string  false  "Filter by status"
// @Param        years    query  string  false  "Filter by student years"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /activitys/{id}/enrollments [get]
func GetEnrollmentByActivityID(c *fiber.Ctx) error {
	id := c.Params("id")
	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// อ่านค่าพารามิเตอร์การแบ่งหน้า
	pagination := models.DefaultPagination()
	if err := c.QueryParser(&pagination); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid pagination parameters"})
	}

	fmt.Println("Pagination:", pagination)

	// รับค่า query param ของ major และ status
	studentMajors := c.Query("majors") // Expecting comma-separated values
	studentStatus := c.Query("status") // Expecting int value
	studentYears := c.Query("years")

	var majorFilter []string
	if studentMajors != "" {
		majorFilter = strings.Split(studentMajors, ",")
	}

	var statusFilter []int
	if studentStatus != "" {
		statusValues := strings.Split(studentStatus, ",")
		for _, val := range statusValues {
			num, err := strconv.Atoi(val)
			if err == nil {
				statusFilter = append(statusFilter, num)
			}
		}
	}

	var studentYearsFilter []int
	if studentYears != "" {
		studentYearsValues := strings.Split(studentYears, ",")
		for _, val := range studentYearsValues {
			num, err := strconv.Atoi(val)
			if err == nil {
				studentYearsFilter = append(studentYearsFilter, num)
			}
		}
	}

	enrollments, total, err := services.GetEnrollmentByActivityID(activityID.Hex(), pagination, majorFilter, statusFilter, studentYearsFilter)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Activity not found",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": enrollments,
		"meta": fiber.Map{
			"currentPage": pagination.Page,
			"perPage":     pagination.Limit,
			"total":       total,
			"totalPages":  (total + int64(pagination.Limit) - 1) / int64(pagination.Limit),
		},
	})
}

// UpdateActivity - อัพเดตข้อมูลกิจกรรม พร้อม ActivityItems
// UpdateActivity - godoc
// @Summary      Update an activity
// @Description  Update an activity
// @Tags         activitys
// @Produce      json
// @Param        id   path  string  true  "Activity ID"
// @Param        activity  body  models.Activity  true  "Activity object"
// @Success      200  {object}  models.Activity
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /activitys/{id} [put]
func UpdateActivity(c *fiber.Ctx) error {
	id := c.Params("id")

	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var request models.Activity
	// ✅ แปลง JSON เป็น struct
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}
	fmt.Println(request)
	// ✅ อัปเดต Activity และ ActivityItems
	updatedActivity, err := services.UpdateActivity(activityID, request)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": updatedActivity,
	})
}

// DeleteActivity - ลบกิจกรรม พร้อม ActivityItems ที่เกี่ยวข้อง
// DeleteActivity - godoc
// @Summary      Delete an activity
// @Description  Delete an activity
// @Tags         activitys
// @Produce      json
// @Param        id   path  string  true  "Activity ID"
// @Success      200
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /activitys/{id} [delete]
func DeleteActivity(c *fiber.Ctx) error {
	id := c.Params("id")
	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// ลบ Activity พร้อม ActivityItems ที่เกี่ยวข้อง
	err = services.DeleteActivity(activityID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Activity and related ActivityItems deleted successfully"})
}
