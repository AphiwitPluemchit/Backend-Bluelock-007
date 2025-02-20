package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateUser(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateUser(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully",
		"user":    user,
	})
}

// GetUsers - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetUsers(c *fiber.Ctx) error {
	users, err := services.GetAllUsers()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching users",
		})
	}

	return c.JSON(users)
}

// GetUserByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetUserByID(c *fiber.Ctx) error {
	id := c.Params("id")
	user, err := services.GetUserByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(user)
}

// UpdateUser - อัปเดตข้อมูลผู้ใช้
func UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	var user models.User

	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateUser(id, &user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating user",
		})
	}

	return c.JSON(fiber.Map{
		"message": "User updated successfully",
	})
}

// DeleteUser - ลบผู้ใช้
func DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteUser(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting user",
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}
