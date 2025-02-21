package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

// CreateActivity - สร้างกิจกรรมใหม่
func CreateActivity(c *fiber.Ctx) error {
	var activity models.Activity

	// แปลง JSON เป็น struct
	if err := c.BodyParser(&activity); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// บันทึก Activity + Items
	err := services.CreateActivity(activity)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create activity and items",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Activity and items created successfully",
	})
}

// GetActivitys - ดึงกิจกรรมทั้งหมด
func GetActivitys(c *fiber.Ctx) error {
	activitys, err := services.GetAllActivitys()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching activitys",
		})
	}

	return c.JSON(activitys)
}

// GetActivityByID - ดึงข้อมูลกิจกรรมตาม ID
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

// UpdateActivity - อัปเดตข้อมูลกิจกรรม
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

// DeleteActivity - ลบกิจกรรม
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
