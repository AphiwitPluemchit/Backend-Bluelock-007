package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateFoodVote(c *fiber.Ctx) error {
	var foodVote models.FoodVote
	if err := c.BodyParser(&foodVote); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateFoodVote(&foodVote)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating foodVote",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "FoodVote created successfully",
		"foodVote": foodVote,
	})
}

// GetFoodVotes - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetFoodVotes(c *fiber.Ctx) error {
	foodVotes, err := services.GetAllFoodVotes()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching foodVotes",
		})
	}

	return c.JSON(foodVotes)
}

// GetFoodVoteByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetFoodVoteByID(c *fiber.Ctx) error {
	id := c.Params("id")
	foodVote, err := services.GetFoodVoteByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "FoodVote not found",
		})
	}

	return c.JSON(foodVote)
}

// UpdateFoodVote - อัปเดตข้อมูลผู้ใช้
func UpdateFoodVote(c *fiber.Ctx) error {
	id := c.Params("id")
	var foodVote models.FoodVote

	if err := c.BodyParser(&foodVote); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateFoodVote(id, &foodVote)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating foodVote",
		})
	}

	return c.JSON(fiber.Map{
		"message": "FoodVote updated successfully",
	})
}

// DeleteFoodVote - ลบผู้ใช้
func DeleteFoodVote(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteFoodVote(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting foodVote",
		})
	}

	return c.JSON(fiber.Map{
		"message": "FoodVote deleted successfully",
	})
}
