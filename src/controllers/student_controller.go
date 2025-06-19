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

	// รับข้อมูลจาก body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	// สำหรับเก็บ error ที่อาจเกิดขึ้น
	var failed []string

	// Loop เพื่อสร้าง Student ทีละคน
	for _, studentData := range req {
		student := models.Student{
			Code:      studentData.Code,
			Name:      studentData.Name,
			EngName:   studentData.EngName,
			Email:     studentData.Code + "@go.buu.ac.th",                            // auto-generate email
			Password:  studentData.Password,                                          // default password
			Status:    calculateStatus(studentData.SoftSkill, studentData.HardSkill), // default status
			SoftSkill: studentData.SoftSkill,                                         // ← ดึงจาก req
			HardSkill: studentData.HardSkill,                                         // ← ดึงจาก req
			Major:     studentData.Major,
		}

		// เรียกใช้ service เพื่อสร้าง student
		err := students.CreateStudent(&student)
		if err != nil {
			failed = append(failed, student.Code) // เก็บรหัสนิสิตที่สร้างไม่สำเร็จ
		}
	}

	// ถ้าล้มเหลวในการสร้างบางคน
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
	log.Println(studentStatus)
	log.Println(majors)
	log.Println(studentYears)
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
	var student models.Student

	if err := c.BodyParser(&student); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := students.UpdateStudent(id, &student)
	if err != nil {
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
	err := students.DeleteStudent(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting student",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Student deleted successfully",
	})
}

// UpdateStudentStatus godoc
// @Summary Update student status to 0
// @Description Set status of students to 0 by IDs
// @Tags students
// @Accept json
// @Produce json
// @Param ids body []map[string]string true "List of student IDs"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /students/status [patch]
func UpdateStudentStatus(c *fiber.Ctx) error {
	var req []struct {
		ID string `json:"id"`
	}

	// รับข้อมูลจาก body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	// วนลูปอัพเดตสถานะของนิสิต
	for _, studentData := range req {
		// เรียกใช้ Service เพื่อเปลี่ยนสถานะของนิสิต
		err := students.UpdateStatusToZero(studentData.ID)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Student status updated to 0 successfully",
	})
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
