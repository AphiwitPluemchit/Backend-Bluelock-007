package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateFormEvaluation(c *fiber.Ctx) error {
	var formEvaluation models.FormEvaluation
	if err := c.BodyParser(&formEvaluation); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateFormEvaluation(&formEvaluation)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating formEvaluation",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":        "FormEvaluation created successfully",
		"formEvaluation": formEvaluation,
	})
}

// GetFormEvaluations - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetFormEvaluations(c *fiber.Ctx) error {
	formEvaluations, err := services.GetAllFormEvaluations()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching formEvaluations",
		})
	}

	return c.JSON(formEvaluations)
}

// GetFormEvaluationByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetFormEvaluationByID(c *fiber.Ctx) error {
	id := c.Params("id")
	formEvaluation, err := services.GetFormEvaluationByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "FormEvaluation not found",
		})
	}

	return c.JSON(formEvaluation)
}

// UpdateFormEvaluation - อัปเดตข้อมูลผู้ใช้
func UpdateFormEvaluation(c *fiber.Ctx) error {
	id := c.Params("id")
	var formEvaluation models.FormEvaluation

	if err := c.BodyParser(&formEvaluation); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateFormEvaluation(id, &formEvaluation)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating formEvaluation",
		})
	}

	return c.JSON(fiber.Map{
		"message": "FormEvaluation updated successfully",
	})
}

// DeleteFormEvaluation - ลบผู้ใช้
func DeleteFormEvaluation(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteFormEvaluation(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting formEvaluation",
		})
	}

	return c.JSON(fiber.Map{
		"message": "FormEvaluation deleted successfully",
	})
}
