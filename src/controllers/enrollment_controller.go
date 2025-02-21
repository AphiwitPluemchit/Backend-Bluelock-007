package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateEnrollment(c *fiber.Ctx) error {
	var enrollment models.Enrollment
	if err := c.BodyParser(&enrollment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateEnrollment(&enrollment)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating enrollment",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "Enrollment created successfully",
		"enrollment": enrollment,
	})
}

// GetEnrollments - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetEnrollments(c *fiber.Ctx) error {
	enrollments, err := services.GetAllEnrollments()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching enrollments",
		})
	}

	return c.JSON(enrollments)
}

// GetEnrollmentByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetEnrollmentByID(c *fiber.Ctx) error {
	id := c.Params("id")
	enrollment, err := services.GetEnrollmentByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Enrollment not found",
		})
	}

	return c.JSON(enrollment)
}

// UpdateEnrollment - อัปเดตข้อมูลผู้ใช้
func UpdateEnrollment(c *fiber.Ctx) error {
	id := c.Params("id")
	var enrollment models.Enrollment

	if err := c.BodyParser(&enrollment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateEnrollment(id, &enrollment)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating enrollment",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Enrollment updated successfully",
	})
}

// DeleteEnrollment - ลบผู้ใช้
func DeleteEnrollment(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteEnrollment(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting enrollment",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Enrollment deleted successfully",
	})
}
