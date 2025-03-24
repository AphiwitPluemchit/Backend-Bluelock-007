package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ✅ CreateStudent - เพิ่ม Student
func CreateStudent(c *fiber.Ctx) error {
	var req struct {
		Name    string `json:"name"`
		EngName string `json:"engName"`
		Code    string `json:"code"`
		Major   string `json:"major"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	student := models.Student{
		Code:      req.Code,
		Name:      req.Name,
		EngName:   req.EngName,
		Email:     req.Code + "@go.buu.ac.th", // auto-generate email
		Password:  "123456",                   // default password
		Status:    1,                          // default status
		SoftSkill: 0,
		HardSkill: 0,
		Major:     req.Major,
	}

	err := services.CreateStudent(&student)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "Student created successfully"})
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

// GetStudents - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetStudents(c *fiber.Ctx) error {
	params := models.DefaultPagination()
	params.Page, _ = strconv.Atoi(c.Query("page", strconv.Itoa(params.Page)))
	params.Limit, _ = strconv.Atoi(c.Query("limit", strconv.Itoa(params.Limit)))
	params.Search = c.Query("search", params.Search)
	params.SortBy = c.Query("sortBy", params.SortBy)
	params.Order = c.Query("order", params.Order)

	majors := cleanList(strings.Split(c.Query("major"), ","))
	years := cleanList(strings.Split(c.Query("studentYear"), ","))

	students, total, totalPages, err := services.GetStudentsWithFilter(params, majors, years)
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

// GetStudentByCode - ดึงข้อมูลผู้ใช้ตาม Code
func GetStudentByCode(c *fiber.Ctx) error {
	code := c.Params("code")
	student, err := services.GetStudentByCode(code)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Student not found",
		})
	}

	return c.JSON(student)
}

// UpdateStudent - อัปเดตข้อมูลผู้ใช้
func UpdateStudent(c *fiber.Ctx) error {
	id := c.Params("id")
	var student models.Student

	if err := c.BodyParser(&student); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateStudent(id, &student)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating student",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Student updated successfully",
	})
}

// DeleteStudent - ลบผู้ใช้
func DeleteStudent(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteStudent(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting student",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Student deleted successfully",
	})
}
