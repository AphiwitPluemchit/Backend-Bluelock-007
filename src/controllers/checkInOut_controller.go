package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateCheckInOut(c *fiber.Ctx) error {
	var checkInOut models.CheckInOut
	if err := c.BodyParser(&checkInOut); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateCheckInOut(&checkInOut)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating checkInOut",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "CheckInOut created successfully",
		"checkInOut": checkInOut,
	})
}

// GetCheckInOuts - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetCheckInOuts(c *fiber.Ctx) error {
	checkInOuts, err := services.GetAllCheckInOuts()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching checkInOuts",
		})
	}

	return c.JSON(checkInOuts)
}

// GetCheckInOutByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetCheckInOutByID(c *fiber.Ctx) error {
	id := c.Params("id")
	checkInOut, err := services.GetCheckInOutByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "CheckInOut not found",
		})
	}

	return c.JSON(checkInOut)
}

// UpdateCheckInOut - อัปเดตข้อมูลผู้ใช้
func UpdateCheckInOut(c *fiber.Ctx) error {
	id := c.Params("id")
	var checkInOut models.CheckInOut

	if err := c.BodyParser(&checkInOut); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateCheckInOut(id, &checkInOut)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating checkInOut",
		})
	}

	return c.JSON(fiber.Map{
		"message": "CheckInOut updated successfully",
	})
}

// DeleteCheckInOut - ลบผู้ใช้
func DeleteCheckInOut(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteCheckInOut(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting checkInOut",
		})
	}

	return c.JSON(fiber.Map{
		"message": "CheckInOut deleted successfully",
	})
}
