package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// ✅ CreateStudent - เพิ่ม Student
func CreateStudent(c *fiber.Ctx) error {
	var req struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Password  string `json:"password"`
		Status    string `json:"status"`
		SoftSkill int    `json:"softSkill"`
		HardSkill int    `json:"hardSkill"`
		Major     string `json:"major"`
	}

	// 1️⃣ ดึงค่าจาก Body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	// 3️⃣ สร้าง Student ใหม่
	student := models.Student{
		Code:      req.Code,
		Name:      req.Name,
		Email:     req.Email,
		Password:  req.Password, // จะถูกเข้ารหัสใน Service
		Status:    req.Status,
		SoftSkill: req.SoftSkill,
		HardSkill: req.HardSkill,
		Major:     req.Major,
	}

	// 4️⃣ เรียกใช้ Service เพื่อบันทึกข้อมูล
	err := services.CreateStudent(&student)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	// 5️⃣ ตอบกลับเมื่อสำเร็จ
	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "Student created successfully"})
}

// GetStudents - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetStudents(c *fiber.Ctx) error {
	students, err := services.GetAllStudents()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching students",
		})
	}

	return c.JSON(students)
}

// GetStudentByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetStudentByID(c *fiber.Ctx) error {
	id := c.Params("id")
	student, err := services.GetStudentByID(id)
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
