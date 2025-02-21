package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func CreateAdmin(c *fiber.Ctx) error {
	var admin models.Admin
	if err := c.BodyParser(&admin); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateAdmin(&admin)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating admin",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Admin created successfully",
		"admin":   admin,
	})
}

// GetAdmins - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAdmins(c *fiber.Ctx) error {
	admins, err := services.GetAllAdmins()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching admins",
		})
	}

	return c.JSON(admins)
}

// GetAdminByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetAdminByID(c *fiber.Ctx) error {
	id := c.Params("id")
	admin, err := services.GetAdminByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Admin not found",
		})
	}

	return c.JSON(admin)
}

// UpdateAdmin - อัปเดตข้อมูลผู้ใช้
func UpdateAdmin(c *fiber.Ctx) error {
	id := c.Params("id")
	var admin models.Admin

	if err := c.BodyParser(&admin); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateAdmin(id, &admin)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating admin",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Admin updated successfully",
	})
}

// DeleteAdmin - ลบผู้ใช้
func DeleteAdmin(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteAdmin(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting admin",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Admin deleted successfully",
	})
}
