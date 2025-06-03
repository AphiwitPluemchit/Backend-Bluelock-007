package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"
	"Backend-Bluelock-007/src/utils"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var path = "./uploads/activity/images/"

// CreateActivity godoc
// @Summary      Create a new activity
// @Description  Create a new activity
// @Tags         activitys
// @Accept       json
// @Produce      json
// @Param        body body models.ActivityDto true "Activity and ActivityItems"
// @Success      201  {object}  models.Activity
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /activitys [post]
// CreateActivity - สร้างกิจกรรมใหม่
func CreateActivity(c *fiber.Ctx) error {
	var request models.ActivityDto

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

// UploadActivityImage godoc
// @Summary      Upload an image for an activity
// @Description  Upload an image for an activity
// @Tags         activitys
// @Accept       multipart/form-data
// @Produce      json
// @Param        id path string true "Activity ID"
// @Param        filename query string false "File name"
// @Param        file formData file true "Image file"
// @Success      200
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /activitys/{id}/image [post]
func UploadActivityImage(c *fiber.Ctx) error {
	id := c.Params("id")
	fileName := c.Query("filename")

	file, err := c.FormFile("file")
	if err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Failed to upload file: "+err.Error())
	}

	// if fileName != ""  then delete old file
	if fileName != "" {
		// 🔥 ลบไฟล์ที่อัปโหลดหากเกิดข้อผิดพลาด
		removeErr := os.Remove(path + fileName)
		if removeErr != nil {
			log.Println("Failed to remove uploaded file:", removeErr)
		}
	}

	fileName = fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(file.Filename))
	filePath := fmt.Sprintf(path+"%s", fileName)
	// folder not exist, create it
	// Create directory if it does not exist
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Println("Failed to create directory:", err)
		// You may want to return an error here instead of continuing
	}

	c.SaveFile(file, filePath)

	//

	// ✅ อัปเดต MongoDB ให้เก็บ Path ไฟล์ที่อัปโหลด

	err = services.UploadActivityImage(id, fileName)
	if err != nil {

		// 🔥 ลบไฟล์ที่อัปโหลดหากเกิดข้อผิดพลาด
		removeErr := os.Remove(filePath)
		if removeErr != nil {
			log.Println("Failed to remove uploaded file:", removeErr)
		}

		return utils.HandleError(c, fiber.StatusInternalServerError, "Failed to update MongoDB: "+err.Error())

	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "File uploaded", "file": filePath})
}

// DeleteActivityImage godoc
// @Summary      Delete an image for an activity
// @Description  Delete an image for an activity
// @Tags         activitys
// @Produce      json
// @Param        id path string true "Activity ID"
// @Param        filename query string false "File name"
// @Success      200
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /activitys/{id}/image [delete]
func DeleteActivityImage(c *fiber.Ctx) error {
	id := c.Params("id")
	fileName := c.Query("filename")

	removeErr := os.Remove(path + fileName)
	if removeErr != nil {
		log.Println("Failed to remove uploaded file:", removeErr)
		return utils.HandleError(c, fiber.StatusInternalServerError, "Failed to remove uploaded file: "+removeErr.Error())
	}

	// Update activity file name to empty string
	err := services.UploadActivityImage(id, "")
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, "Failed to update MongoDB: "+err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "File deleted"})
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
// @Param        skills          query  string  false  "Filter by skill"
// @Param        activityStates  query  string  false  "Filter by activityState"
// @Param        majors          query  string  false  "Filter by major"
// @Param        studentYears    query  string  false  "Filter by studentYear"
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
	skills := c.Query("skills")                 // เช่น skill=soft,hard
	activityStates := c.Query("activityStates") // เช่น activityState=open,planning
	majors := c.Query("majors")                 // เช่น major=CS,SE
	studentYears := c.Query("studentYears")     // เช่น studentYear=1,2,3

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
// @Param        id   path  string  true  "ActivityItem ID"
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
func GetEnrollmentByActivityItemID(c *fiber.Ctx) error {
	activityItemID := c.Params("id")
	itemID, err := primitive.ObjectIDFromHex(activityItemID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// อ่านค่าพารามิเตอร์การแบ่งหน้า
	pagination := models.DefaultPagination()
	if err := c.QueryParser(&pagination); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid pagination parameters"})
	}
	log.Println(pagination)
	// รับค่า query param
	studentMajors := c.Query("majors")
	studentStatus := c.Query("studentStatus")
	studentYears := c.Query("studentYear")

	var majorFilter []string
	if studentMajors != "" {
		majorFilter = strings.Split(studentMajors, ",")
	}

	var statusFilter []int
	if studentStatus != "" {
		statusValues := strings.Split(studentStatus, ",")
		for _, val := range statusValues {
			if num, err := strconv.Atoi(val); err == nil {
				statusFilter = append(statusFilter, num)
			}
		}
	}

	var studentYearsFilter []int
	if studentYears != "" {
		studentYearsValues := strings.Split(studentYears, ",")
		for _, val := range studentYearsValues {
			if num, err := strconv.Atoi(val); err == nil {
				studentYearsFilter = append(studentYearsFilter, num)
			}
		}
	}
	log.Println(majorFilter)
	log.Println(statusFilter)
	log.Println(studentYearsFilter)
	student, total, err := services.GetEnrollmentByActivityItemID(itemID, pagination, majorFilter, statusFilter, studentYearsFilter)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "ActivityItem not found",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": student,
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

	var request models.ActivityDto
	// ✅ แปลง JSON เป็น struct
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

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

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Activity and related ActivityItems were deleted "})
}
