package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateMajor(c *fiber.Ctx) error {
	var major models.Major
	if err := c.BodyParser(&major); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateMajor(&major)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating major",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Major created successfully",
		"major":   major,
	})
}

// GetMajors - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetMajors(c *fiber.Ctx) error {
	majors, err := services.GetAllMajors()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching majors",
		})
	}

	return c.JSON(majors)
}

// GetMajorByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetMajorByID(c *fiber.Ctx) error {
	id := c.Params("id")
	major, err := services.GetMajorByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Major not found",
		})
	}

	return c.JSON(major)
}

// UpdateMajor - อัปเดตข้อมูลผู้ใช้
func UpdateMajor(c *fiber.Ctx) error {
	id := c.Params("id")
	var major models.Major

	if err := c.BodyParser(&major); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateMajor(id, &major)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating major",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Major updated successfully",
	})
}

// DeleteMajor - ลบผู้ใช้
func DeleteMajor(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteMajor(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting major",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Major deleted successfully",
	})
}
