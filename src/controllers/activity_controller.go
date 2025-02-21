package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateActivity(c *fiber.Ctx) error {
	var activity models.Activity
	if err := c.BodyParser(&activity); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateActivity(&activity)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating activity",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "Activity created successfully",
		"activity": activity,
	})
}

// GetActivitys - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetActivitys(c *fiber.Ctx) error {
	activitys, err := services.GetAllActivitys()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching activitys",
		})
	}

	return c.JSON(activitys)
}

// GetActivityByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetActivityByID(c *fiber.Ctx) error {
	id := c.Params("id")
	activity, err := services.GetActivityByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Activity not found",
		})
	}

	return c.JSON(activity)
}

// UpdateActivity - อัปเดตข้อมูลผู้ใช้
func UpdateActivity(c *fiber.Ctx) error {
	id := c.Params("id")
	var activity models.Activity

	if err := c.BodyParser(&activity); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateActivity(id, &activity)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating activity",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Activity updated successfully",
	})
}

// DeleteActivity - ลบผู้ใช้
func DeleteActivity(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteActivity(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting activity",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Activity deleted successfully",
	})
}
