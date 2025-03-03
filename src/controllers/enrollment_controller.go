package controllers

import (
	"Backend-Bluelock-007/src/services"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ✅ 1. Student ลงทะเบียนกิจกรรม
func CreateEnrollment(c *fiber.Ctx) error {
	var req struct {
		ActivityItemID string `json:"activityItemId"`
		StudentID      string `json:"studentId"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	activityItemID, _ := primitive.ObjectIDFromHex(req.ActivityItemID)
	studentID, _ := primitive.ObjectIDFromHex(req.StudentID)

	err := services.RegisterStudent(activityItemID, studentID)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "Enrollment successful"})
}

// ✅ 2. Student ดูกิจกรรมที่ลงทะเบียนไปแล้ว
func GetEnrollmentsByStudent(c *fiber.Ctx) error {
	studentID, err := primitive.ObjectIDFromHex(c.Params("studentId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid studentId format"})
	}

	enrollments, err := services.GetEnrollmentsByStudent(studentID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(enrollments)
}

// ✅ 3. Student ยกเลิกการลงทะเบียน
func DeleteEnrollment(c *fiber.Ctx) error {
	activityItemID, _ := primitive.ObjectIDFromHex(c.Params("activityItemId"))
	studentID, _ := primitive.ObjectIDFromHex(c.Params("studentId"))

	err := services.UnregisterStudent(activityItemID, studentID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Enrollment deleted"})
}

// ✅ 4. Admin ดู Student ที่ลงทะเบียนในกิจกรรม
func GetStudentsByActivity(c *fiber.Ctx) error {
	activityItemID, err := primitive.ObjectIDFromHex(c.Params("activityItemId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid activityItemId format"})
	}

	enrollmentData, err := services.GetStudentsByActivity(activityItemID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(enrollmentData)
}

// ✅ 5. Student ดูกิจกรรมที่ลงทะเบียนไว้ (1 ตัว)
func GetEnrollmentByStudentAndActivity(c *fiber.Ctx) error {
	studentID, _ := primitive.ObjectIDFromHex(c.Params("studentId"))
	activityItemID, _ := primitive.ObjectIDFromHex(c.Params("activityItemId"))

	enrollment, err := services.GetEnrollmentByStudentAndActivity(studentID, activityItemID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Enrollment not found"})
	}

	return c.JSON(enrollment)
}
