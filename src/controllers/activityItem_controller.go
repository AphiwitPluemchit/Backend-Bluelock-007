package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateActivityItem(c *fiber.Ctx) error {
	var activityItem models.ActivityItem
	if err := c.BodyParser(&activityItem); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateActivityItem(&activityItem)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating activityItem",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":      "ActivityItem created successfully",
		"activityItem": activityItem,
	})
}

// GetActivityItems - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetActivityItems(c *fiber.Ctx) error {
	activityItems, err := services.GetAllActivityItems()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching activityItems",
		})
	}

	return c.JSON(activityItems)
}

// GetActivityItemByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetActivityItemByID(c *fiber.Ctx) error {
	id := c.Params("id")
	activityItem, err := services.GetActivityItemByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "ActivityItem not found",
		})
	}

	return c.JSON(activityItem)
}

// UpdateActivityItem - อัปเดตข้อมูลผู้ใช้
func UpdateActivityItem(c *fiber.Ctx) error {
	id := c.Params("id")
	var activityItem models.ActivityItem

	if err := c.BodyParser(&activityItem); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateActivityItem(id, &activityItem)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating activityItem",
		})
	}

	return c.JSON(fiber.Map{
		"message": "ActivityItem updated successfully",
	})
}

// DeleteActivityItem - ลบผู้ใช้
func DeleteActivityItem(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteActivityItem(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting activityItem",
		})
	}

	return c.JSON(fiber.Map{
		"message": "ActivityItem deleted successfully",
	})
}
