package controllers

import (
	"Backend-Bluelock-007/src/services"
	"Backend-Bluelock-007/src/utils"

	"github.com/gofiber/fiber/v2"
)

// LoginUser - สำหรับ login ทั้ง student และ admin
func LoginUser(c *fiber.Ctx) error {
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	user, err := services.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	// ⏩ สร้าง JWT
	token, err := utils.GenerateJWT(user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Token creation failed"})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user": fiber.Map{
			"id":    user.RefID.Hex(),
			"name":  user.Name,
			"email": user.Email,
			"code":  user.Code,
			"role":  user.Role,
		},
	})
}
