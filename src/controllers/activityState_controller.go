package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateActivityState(c *fiber.Ctx) error {
	var activity models.ActivityState
	if err := c.BodyParser(&activity); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateActivityState(&activity)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating activity",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "ActivityState created successfully",
		"activity": activity,
	})
}

// GetActivityStates - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetActivityStates(c *fiber.Ctx) error {
	activitys, err := services.GetAllActivityStates()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching activitys",
		})
	}

	return c.JSON(activitys)
}

// GetActivityStateByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetActivityStateByID(c *fiber.Ctx) error {
	id := c.Params("id")
	activity, err := services.GetActivityStateByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "ActivityState not found",
		})
	}

	return c.JSON(activity)
}

// UpdateActivityState - อัปเดตข้อมูลผู้ใช้
func UpdateActivityState(c *fiber.Ctx) error {
	id := c.Params("id")
	var activity models.ActivityState

	if err := c.BodyParser(&activity); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateActivityState(id, &activity)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating activity",
		})
	}

	return c.JSON(fiber.Map{
		"message": "ActivityState updated successfully",
	})
}

// DeleteActivityState - ลบผู้ใช้
func DeleteActivityState(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteActivityState(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting activity",
		})
	}

	return c.JSON(fiber.Map{
		"message": "ActivityState deleted successfully",
	})
}
