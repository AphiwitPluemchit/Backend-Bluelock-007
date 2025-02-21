package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateEvaluationScore(c *fiber.Ctx) error {
	var evaluationScore models.EvaluationScore
	if err := c.BodyParser(&evaluationScore); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateEvaluationScore(&evaluationScore)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating evaluationScore",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":         "EvaluationScore created successfully",
		"evaluationScore": evaluationScore,
	})
}

// GetEvaluationScores - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetEvaluationScores(c *fiber.Ctx) error {
	evaluationScores, err := services.GetAllEvaluationScores()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching evaluationScores",
		})
	}

	return c.JSON(evaluationScores)
}

// GetEvaluationScoreByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetEvaluationScoreByID(c *fiber.Ctx) error {
	id := c.Params("id")
	evaluationScore, err := services.GetEvaluationScoreByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "EvaluationScore not found",
		})
	}

	return c.JSON(evaluationScore)
}

// UpdateEvaluationScore - อัปเดตข้อมูลผู้ใช้
func UpdateEvaluationScore(c *fiber.Ctx) error {
	id := c.Params("id")
	var evaluationScore models.EvaluationScore

	if err := c.BodyParser(&evaluationScore); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateEvaluationScore(id, &evaluationScore)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating evaluationScore",
		})
	}

	return c.JSON(fiber.Map{
		"message": "EvaluationScore updated successfully",
	})
}

// DeleteEvaluationScore - ลบผู้ใช้
func DeleteEvaluationScore(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteEvaluationScore(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting evaluationScore",
		})
	}

	return c.JSON(fiber.Map{
		"message": "EvaluationScore deleted successfully",
	})
}
