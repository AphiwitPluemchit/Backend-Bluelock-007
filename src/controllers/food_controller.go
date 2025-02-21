package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateFood(c *fiber.Ctx) error {
	var food models.Food
	if err := c.BodyParser(&food); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateFood(&food)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating food",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Food created successfully",
		"food":    food,
	})
}

// GetFoods - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetFoods(c *fiber.Ctx) error {
	foods, err := services.GetAllFoods()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching foods",
		})
	}

	return c.JSON(foods)
}

// GetFoodByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetFoodByID(c *fiber.Ctx) error {
	id := c.Params("id")
	food, err := services.GetFoodByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Food not found",
		})
	}

	return c.JSON(food)
}

// UpdateFood - อัปเดตข้อมูลผู้ใช้
func UpdateFood(c *fiber.Ctx) error {
	id := c.Params("id")
	var food models.Food

	if err := c.BodyParser(&food); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateFood(id, &food)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating food",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Food updated successfully",
	})
}

// DeleteFood - ลบผู้ใช้
func DeleteFood(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteFood(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting food",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Food deleted successfully",
	})
}
