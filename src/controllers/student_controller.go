package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/students"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// CreateStudent godoc
// @Summary Create students
// @Description Create one or more students
// @Tags students
// @Accept json
// @Produce json
// @Param students body []models.Student true "List of students to create"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /students [post]
// ✅ CreateStudent - เพิ่ม Student หลายคน
func CreateStudent(c *fiber.Ctx) error {
	var req []struct {
		Name      string `json:"name"`
		EngName   string `json:"engName"`
		Code      string `json:"code"`
		Major     string `json:"major"`
		Password  string `json:"password"`
		SoftSkill int    `json:"softSkill"`
		HardSkill int    `json:"hardSkill"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	var failed []string

	for _, studentData := range req {
		// 👉 1. เตรียม Student profile
		student := models.Student{
			Code:      studentData.Code,
			Name:      studentData.Name,
			EngName:   studentData.EngName,
			Status:    calculateStatus(studentData.SoftSkill, studentData.HardSkill),
			SoftSkill: studentData.SoftSkill,
			HardSkill: studentData.HardSkill,
			Major:     studentData.Major,
		}

		// 👉 2. เตรียม User auth
		user := models.User{
			Email:    strings.ToLower(studentData.Code + "@go.buu.ac.th"),
			Password: studentData.Password,
		}

		// 👉 3. สร้างผ่าน service (จะเชื่อม refId ให้ภายใน)
		err := students.CreateStudent(&user, &student)
		if err != nil {
			log.Println("❌ Failed to create student:", student.Code, err)
			failed = append(failed, student.Code)
		}
	}

	if len(failed) > 0 {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error":  "Failed to create some students",
			"failed": failed,
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "Students created successfully",
	})
}

func cleanList(arr []string) []string {
	var result []string
	for _, v := range arr {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

// GetStudents godoc
// @Summary Get students
// @Description Get all students with optional filters
// @Tags students
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Page size"
// @Param search query string false "Search keyword"
// @Param sortBy query string false "Sort by field"
// @Param order query string false "Order (asc/desc)"
// @Param studentStatus query string false "Student status (comma separated)"
// @Param major query string false "Major (comma separated)"
// @Param studentYear query string false "Student year (comma separated)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /students [get]
func GetStudents(c *fiber.Ctx) error {
	params := models.DefaultPagination()
	params.Page, _ = strconv.Atoi(c.Query("page", strconv.Itoa(params.Page)))
	params.Limit, _ = strconv.Atoi(c.Query("limit", strconv.Itoa(params.Limit)))
	params.Search = c.Query("search", params.Search)
	params.SortBy = c.Query("sortBy", params.SortBy)
	params.Order = c.Query("order", params.Order)

	studentStatus := cleanList(strings.Split(c.Query("studentStatus"), ","))
	majors := cleanList(strings.Split(c.Query("major"), ","))
	studentYears := cleanList(strings.Split(c.Query("studentYear"), ","))
	studentCode := cleanList(strings.Split(c.Query("studentCode"), ","))
	log.Println("studentStatus", studentStatus)
	log.Println("majors", majors)
	log.Println("studentYears", studentYears)
	log.Println("studentCode", studentCode)
	students, total, totalPages, err := students.GetStudentsWithFilter(params, majors, studentYears, studentStatus)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching students"})
	}

	return c.JSON(fiber.Map{
		"data": students,
		"meta": fiber.Map{
			"page":       params.Page,
			"limit":      params.Limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

// GetStudentByCode godoc
// @Summary Get student by code
// @Description Get a student by their code
// @Tags students
// @Accept json
// @Produce json
// @Param code path string true "Student code"
// @Success 200 {object} models.Student
// @Failure 404 {object} map[string]interface{}
// @Router /students/{code} [get]
func GetStudentByCode(c *fiber.Ctx) error {
	code := c.Params("code")
	student, err := students.GetStudentByCode(code)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Student not found",
		})
	}

	return c.JSON(student)
}

// UpdateStudent godoc
// @Summary Update student
// @Description Update a student's information
// @Tags students
// @Accept json
// @Produce json
// @Param id path string true "Student ID"
// @Param student body models.Student true "Student data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /students/{id} [put]
func UpdateStudent(c *fiber.Ctx) error {
	id := c.Params("id")

	// ✅ struct แยก สำหรับรับค่าจาก frontend
	var req struct {
		Name      string `json:"name"`
		EngName   string `json:"engName"`
		Code      string `json:"code"`
		Major     string `json:"major"`
		SoftSkill int    `json:"softSkill"`
		HardSkill int    `json:"hardSkill"`
		Email     string `json:"email"` // ✅ เพิ่ม email
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// ✅ map เข้า model.Student
	student := &models.Student{
		Name:      req.Name,
		EngName:   req.EngName,
		Code:      req.Code,
		Major:     req.Major,
		SoftSkill: req.SoftSkill,
		HardSkill: req.HardSkill,
	}

	// ✅ ส่งทั้ง student และ email แยกไป
	if err := students.UpdateStudent(id, student, req.Email); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating student",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Student updated successfully",
	})
}

// DeleteStudent godoc
// @Summary Delete student
// @Description Delete a student by ID
// @Tags students
// @Accept json
// @Produce json
// @Param id path string true "Student ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /students/{id} [delete]
func DeleteStudent(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := students.DeleteStudent(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting student",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Student deleted successfully",
	})
}

// UpdateStudentStatusByIDs - อัปเดตสถานะนักเรียนหลายคนโดยใช้ ID
func UpdateStudentStatusByIDs(c *fiber.Ctx) error {
	type UpdateStatusRequest struct {
		StudentIDs []string `json:"studentIds"`
		Status     int      `json:"status"`
	}

	var req UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	if len(req.StudentIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Student IDs are required",
			"code":  "MISSING_IDS",
		})
	}

	err := students.UpdateStudentStatusByIDs(req.StudentIDs, req.Status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update student status",
			"code":  "UPDATE_FAILED",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Student status updated successfully",
		"updated": len(req.StudentIDs),
		"success": true,
	})
}

func GetSammaryByCode(c *fiber.Ctx) error {
	code := c.Params("code")
	student, err := students.GetSammaryByCode(code)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Student not found"})
	}
	return c.JSON(student)
}
func calculateStatus(softSkill, hardSkill int) int {
	total := softSkill + hardSkill

	switch {
	case total >= 20:
		return 3 // ครบ
	case total >= 10:
		return 2 // น้อย
	default:
		return 1 // น้อยมาก
	}
}
func GetSammaryAll(c *fiber.Ctx) error {
	majors := cleanList(strings.Split(c.Query("major"), ","))
	studentYears := cleanList(strings.Split(c.Query("studentYear"), ","))
	summary, err := students.GetStudentSummary(majors, studentYears)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error generating summary"})
	}
	return c.JSON(summary)
}
