package controllers

import (
	"Backend-Bluelock-007/src/models"
	programs "Backend-Bluelock-007/src/services/programs"
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

var path = "./uploads/program/images/"

// CreateProgram godoc
// @Summary      Create a new program
// @Description  Create a new program
// @Tags         programs
// @Accept       json
// @Produce      json
// @Param        body body models.ProgramDto true "Program and ProgramItems"
// @Success      201  {object}  models.Program
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs [post]
// CreateProgram - สร้างกิจกรรมใหม่
func CreateProgram(c *fiber.Ctx) error {
	var request models.ProgramDto

	// แปลง JSON เป็น struct
	if err := c.BodyParser(&request); err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid input: "+err.Error())
	}

	// บันทึก Program + Items
	program, err := programs.CreateProgram(&request)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Program and ProgramItems created successfully",
		"data":    program,
	})
}

// UploadProgramImage godoc
// @Summary      Upload an image for an program
// @Description  Upload an image for an program
// @Tags         programs
// @Accept       multipart/form-data
// @Produce      json
// @Param        id path string true "Program ID"
// @Param        filename query string false "File name"
// @Param        file formData file true "Image file"
// @Success      200
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs/{id}/image [post]
func UploadProgramImage(c *fiber.Ctx) error {
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

	err = programs.UploadProgramImage(id, fileName)
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

// DeleteProgramImage godoc
// @Summary      Delete an image for an program
// @Description  Delete an image for an program
// @Tags         programs
// @Produce      json
// @Param        id path string true "Program ID"
// @Param        filename query string false "File name"
// @Success      200
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs/{id}/image [delete]
func DeleteProgramImage(c *fiber.Ctx) error {
	id := c.Params("id")
	fileName := c.Query("filename")

	removeErr := os.Remove(path + fileName)
	if removeErr != nil {
		log.Println("Failed to remove uploaded file:", removeErr)
		return utils.HandleError(c, fiber.StatusInternalServerError, "Failed to remove uploaded file: "+removeErr.Error())
	}

	// Update program file name to empty string
	err := programs.UploadProgramImage(id, "")
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, "Failed to update MongoDB: "+err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "File deleted"})
}

// GetAllPrograms godoc
// @Summary      Get all programs with pagination, search, and sorting
// @Description  Get all programs with pagination, search, and sorting
// @Tags         programs
// @Produce      json
// @Param        page   query  int  false  "Page number" default(1)
// @Param        limit  query  int  false  "Number of items per page" default(10)
// @Param        search query  string  false  "Search term"
// @Param        sortBy query  string  false  "Field to sort by" default(name)
// @Param        order  query  string  false  "Sort order (asc or desc)" default(asc)
// @Param        skills          query  string  false  "Filter by skill"
// @Param        programStates  query  string  false  "Filter by programState"
// @Param        majors          query  string  false  "Filter by major"
// @Param        studentYears    query  string  false  "Filter by studentYear"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs [get]
func GetAllPrograms(c *fiber.Ctx) error {
	// ใช้ DTO Default แล้วอัปเดตค่าจาก Query Parameter
	params := models.DefaultPagination()

	// อ่านค่า Query Parameter และแปลงเป็น int
	params.Page, _ = strconv.Atoi(c.Query("page", strconv.Itoa(params.Page)))
	params.Limit, _ = strconv.Atoi(c.Query("limit", strconv.Itoa(params.Limit)))
	params.Search = c.Query("search", params.Search)
	params.SortBy = c.Query("sortBy", params.SortBy)
	params.Order = c.Query("order", params.Order)

	// ดึงค่าจาก Query Parameters และแปลงเป็น array
	skills := c.Query("skills")               // เช่น skill=soft,hard
	programStates := c.Query("programStates") // เช่น programState=open,planning
	majors := c.Query("majors")               // เช่น major=CS,SE
	studentYears := c.Query("studentYears")   // เช่น studentYear=1,2,3

	// Convert comma-separated values into arrays
	skillFilter := strings.Split(skills, ",")
	stateFilter := strings.Split(programStates, ",")
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
	programs, total, totalPages, err := programs.GetAllPrograms(params, skillFilter, stateFilter, majorFilter, yearFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch programs",
		})
	}

	// ส่ง Response กลับไป
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": programs,
		"meta": fiber.Map{
			"page":       params.Page,
			"limit":      params.Limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

// GetProgramByID - ดึงข้อมูลกิจกรรมตาม ID พร้อม ProgramItems
// GetProgramByID - godoc
// @Summary      Get an program by ID
// @Description  Get an program by ID
// @Tags         programs
// @Produce      json
// @Param        id   path  string  true  "Program ID"
// @Success      200  {object}  models.Program
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs/{id} [get]
func GetProgramByID(c *fiber.Ctx) error {
	id := c.Params("id")
	programID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// ดึงข้อมูล Program พร้อม ProgramItems
	program, err := programs.GetProgramByID(programID.Hex())
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Program not found"})
	}

	// ส่งข้อมูลกลับรวมทั้ง ProgramItems
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": program,
	})
}

// GetEnrollmentSummaryByProgramID - ดึงข้อมูลสรุปการลงทะเบียน
// GetEnrollmentSummaryByProgramID - godoc
// @Summary      Get enrollment summary by program ID
// @Description  Get enrollment summary by program ID
// @Tags         programs
// @Produce      json
// @Param        id   path  string  true  "Program ID"
// @Success      200  {object} 	models.EnrollmentSummary
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs/{id}/enrollment-summary [get]
func GetEnrollmentSummaryByProgramID(c *fiber.Ctx) error {
	id := c.Params("id")
	programID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// ดึงข้อมูลสรุปการลงทะเบียน
	enrollmentSummary, err := programs.GetProgramEnrollSummary(programID.Hex())
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Program not found",
			"message": err.Error(),
		})
	}

	// ส่งข้อมูลกลับ
	return c.Status(fiber.StatusOK).JSON(enrollmentSummary)
}

// GetEnrollmentByProgramID - ดึงข้อมูลการลงทะเบียนตาม ID กิจกรรม
// GetEnrollmentByProgramID - godoc
// @Summary      Get enrollments by program ID
// @Description  Get enrollments by program ID
// @Tags         programs
// @Produce      json
// @Param        id   path  string  true  "ProgramItem ID"
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
// @Router       /programs/{id}/enrollments [get]
func GetEnrollmentByProgramItemID(c *fiber.Ctx) error {
	programItemID := c.Params("id")
	itemID, err := primitive.ObjectIDFromHex(programItemID)
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
	studentMajors := c.Query("major")
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
	student, total, err := programs.GetEnrollmentByProgramItemID(itemID, pagination, majorFilter, statusFilter, studentYearsFilter)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "ProgramItem not found",
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

// GET /enrollments/by-program/:id
func GetEnrollmentsByProgramID(c *fiber.Ctx) error {
	programID := c.Params("id")
	aID, err := primitive.ObjectIDFromHex(programID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// อ่าน pagination
	pagination := models.DefaultPagination()
	if err := c.QueryParser(&pagination); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid pagination parameters"})
	}

	// ฟิลเตอร์
	studentMajors := c.Query("major")
	studentStatus := c.Query("studentStatus")
	studentYears := c.Query("studentYear")

	var majorFilter []string
	if studentMajors != "" {
		majorFilter = strings.Split(studentMajors, ",")
	}

	var statusFilter []int
	if studentStatus != "" {
		for _, v := range strings.Split(studentStatus, ",") {
			if num, err := strconv.Atoi(v); err == nil {
				statusFilter = append(statusFilter, num)
			}
		}
	}

	var studentYearsFilter []int
	if studentYears != "" {
		for _, v := range strings.Split(studentYears, ",") {
			if num, err := strconv.Atoi(v); err == nil {
				studentYearsFilter = append(studentYearsFilter, num)
			}
		}
	}

	students, total, err := programs.GetEnrollmentsByProgramID(aID, pagination, majorFilter, statusFilter, studentYearsFilter)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Program not found or no program items",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": students,
		"meta": fiber.Map{
			"currentPage": pagination.Page,
			"perPage":     pagination.Limit,
			"total":       total,
			"totalPages":  (total + int64(pagination.Limit) - 1) / int64(pagination.Limit),
		},
	})
}

// UpdateProgram - อัพเดตข้อมูลกิจกรรม พร้อม ProgramItems
// UpdateProgram - godoc
// @Summary      Update an program
// @Description  Update an program
// @Tags         programs
// @Produce      json
// @Param        id   path  string  true  "Program ID"
// @Param        program  body  models.Program  true  "Program object"
// @Success      200  {object}  models.Program
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs/{id} [put]
func UpdateProgram(c *fiber.Ctx) error {
	id := c.Params("id")

	programID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var request models.ProgramDto
	// ✅ แปลง JSON เป็น struct
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// ✅ อัปเดต Program และ ProgramItems
	updatedProgram, err := programs.UpdateProgram(programID, request)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": updatedProgram,
	})
}

// DeleteProgram - ลบกิจกรรม พร้อม ProgramItems ที่เกี่ยวข้อง
// DeleteProgram - godoc
// @Summary      Delete an program
// @Description  Delete an program
// @Tags         programs
// @Produce      json
// @Param        id   path  string  true  "Program ID"
// @Success      200
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs/{id} [delete]
func DeleteProgram(c *fiber.Ctx) error {
	id := c.Params("id")
	programID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// ลบ Program พร้อม ProgramItems ที่เกี่ยวข้อง
	err = programs.DeleteProgram(programID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Program and related ProgramItems were deleted "})
}

// GetAllProgramCalendar - ดึง Program และ ProgramItems ตามเดือนและปีที่ระบุ
// GetAllProgramCalendar - godoc
// @Summary      Get all program calendar
// @Description  Get all program calendar
// @Tags         programs
// @Produce      json
// @Param        month   path  int  true  "Month"
// @Param        year   path  int  true  "Year"
// @Success      200  {object}  []models.ProgramDto
// @Failure      500  {object}  models.ErrorResponse
// @Router       /programs/calendar/{month}/{year} [get]
func GetAllProgramCalendar(c *fiber.Ctx) error {
	month, _ := strconv.Atoi(c.Params("month"))
	year, _ := strconv.Atoi(c.Params("year"))

	calendar, err := programs.GetAllProgramCalendar(month, year)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(calendar)
}
