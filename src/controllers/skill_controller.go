package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateSkill(c *fiber.Ctx) error {
	var skill models.Skill
	if err := c.BodyParser(&skill); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateSkill(&skill)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating skill",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Skill created successfully",
		"skill":   skill,
	})
}

// GetSkills - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetSkills(c *fiber.Ctx) error {
	skills, err := services.GetAllSkills()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching skills",
		})
	}

	return c.JSON(skills)
}

// GetSkillByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetSkillByID(c *fiber.Ctx) error {
	id := c.Params("id")
	skill, err := services.GetSkillByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Skill not found",
		})
	}

	return c.JSON(skill)
}

// UpdateSkill - อัปเดตข้อมูลผู้ใช้
func UpdateSkill(c *fiber.Ctx) error {
	id := c.Params("id")
	var skill models.Skill

	if err := c.BodyParser(&skill); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateSkill(id, &skill)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating skill",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Skill updated successfully",
	})
}

// DeleteSkill - ลบผู้ใช้
func DeleteSkill(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteSkill(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting skill",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Skill deleted successfully",
	})
}
