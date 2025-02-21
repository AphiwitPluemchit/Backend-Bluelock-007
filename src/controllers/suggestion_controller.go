package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateSuggestion(c *fiber.Ctx) error {
	var suggestion models.Suggestion
	if err := c.BodyParser(&suggestion); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateSuggestion(&suggestion)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating suggestion",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "Suggestion created successfully",
		"suggestion": suggestion,
	})
}

// GetSuggestions - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetSuggestions(c *fiber.Ctx) error {
	suggestions, err := services.GetAllSuggestions()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching suggestions",
		})
	}

	return c.JSON(suggestions)
}

// GetSuggestionByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetSuggestionByID(c *fiber.Ctx) error {
	id := c.Params("id")
	suggestion, err := services.GetSuggestionByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Suggestion not found",
		})
	}

	return c.JSON(suggestion)
}

// UpdateSuggestion - อัปเดตข้อมูลผู้ใช้
func UpdateSuggestion(c *fiber.Ctx) error {
	id := c.Params("id")
	var suggestion models.Suggestion

	if err := c.BodyParser(&suggestion); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateSuggestion(id, &suggestion)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating suggestion",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Suggestion updated successfully",
	})
}

// DeleteSuggestion - ลบผู้ใช้
func DeleteSuggestion(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteSuggestion(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting suggestion",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Suggestion deleted successfully",
	})
}
