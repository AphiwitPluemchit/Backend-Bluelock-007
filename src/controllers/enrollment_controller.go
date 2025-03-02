package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateEnrollment - ลงทะเบียนกิจกรรม
func CreateEnrollment(c *fiber.Ctx) error {
	var req struct {
		FoodVoteID     string `json:"foodVoteId"`
		ActivityItemID string `json:"activityItemId"`
		StudentID      string `json:"studentId"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	foodVoteID, err := primitive.ObjectIDFromHex(req.FoodVoteID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid foodVoteId format"})
	}

	activityItemID, err := primitive.ObjectIDFromHex(req.ActivityItemID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid activityItemId format"})
	}

	studentID, err := primitive.ObjectIDFromHex(req.StudentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid studentId format"})
	}

	err = services.RegisterActivityItem(foodVoteID, activityItemID, studentID)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Enrollment created or updated successfully"})
}

// GetAllEnrollments - ดึงข้อมูลทั้งหมด
func GetAllEnrollments(c *fiber.Ctx) error {
	enrollments, err := services.GetAllEnrollments()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(enrollments)
}

// GetEnrollmentByID - ดึงข้อมูลจาก ID
func GetEnrollmentByID(c *fiber.Ctx) error {
	id := c.Params("id")
	enrollment, err := services.GetEnrollmentByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Enrollment not found"})
	}

	return c.JSON(enrollment)
}

// GetEnrollmentsByStudent - ดึงข้อมูลกิจกรรมทั้งหมดที่นิสิตเข้าร่วม
func GetEnrollmentsByStudent(c *fiber.Ctx) error {
	studentID := c.Params("studentId")

	// ตรวจสอบว่า studentId เป็น ObjectID ที่ถูกต้อง
	if !primitive.IsValidObjectID(studentID) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid studentId format"})
	}

	studentObjectID, _ := primitive.ObjectIDFromHex(studentID)

	// เรียกใช้ Service
	enrollments, err := services.GetEnrollmentsByStudent(studentObjectID)
	if err != nil {
		if err.Error() == "no enrollments found for the student" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No enrollments found for the student"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(enrollments)
}

// GetEnrollmentByStudentAndActivity - ดึงข้อมูลกิจกรรมที่นิสิตเลือก
func GetEnrollmentByStudentAndActivity(c *fiber.Ctx) error {
	studentID := c.Params("studentId")
	activityItemID := c.Params("activityItemId")

	// ตรวจสอบ ObjectID
	studentObjectID, err := primitive.ObjectIDFromHex(studentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid studentId format"})
	}

	activityItemObjectID, err := primitive.ObjectIDFromHex(activityItemID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid activityItemId format"})
	}

	// เรียกใช้ Service
	enrollment, err := services.GetEnrollmentByStudentAndActivity(studentObjectID, activityItemObjectID)
	if err != nil {
		if err.Error() == "enrollment not found" || err.Error() == "activity item not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(enrollment)
}

// UpdateEnrollment - อัปเดตข้อมูล
func UpdateEnrollment(c *fiber.Ctx) error {
	id := c.Params("id")
	var enrollment models.Enrollment

	if err := c.BodyParser(&enrollment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	err := services.UpdateEnrollment(id, &enrollment)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error updating enrollment"})
	}

	return c.JSON(fiber.Map{"message": "Enrollment updated successfully"})
}

// DeleteEnrollment - ลบข้อมูล
func DeleteEnrollment(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteEnrollment(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error deleting enrollment"})
	}

	return c.JSON(fiber.Map{"message": "Enrollment deleted successfully"})
}
