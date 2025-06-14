package controllers

import (
	"Backend-Bluelock-007/src/services"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func GenerateLink(c *fiber.Ctx) error {
	var body struct {
		ActivityItemId string `json:"activityItemId"`
		Type           string `json:"type"` // "เข้า" หรือ "ออก"
	}

	if err := c.BodyParser(&body); err != nil || body.ActivityItemId == "" || body.Type == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ activityItemId และ type"})
	}

	// ✅ ดึง userId จาก JWT middleware
	userIdRaw := c.Locals("userId")
	userId, ok := userIdRaw.(string)
	if !ok || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	uuid, err := services.GenerateCheckinUUID(body.ActivityItemId, body.Type, userId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "ไม่สามารถสร้าง UUID ได้"})
	}

	return c.JSON(fiber.Map{
		"uuid": uuid,
		"url":  fmt.Sprintf("/%s/%s", body.Type, uuid),
		"type": body.Type,
	})
}

func Checkin(c *fiber.Ctx) error {
	uuid := c.Params("uuid")

	var body struct {
		UserId string `json:"userId"` // จาก client ที่สแกน
	}
	if err := c.BodyParser(&body); err != nil || body.UserId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ userId"})
	}

	success, msg := services.Checkin(uuid, body.UserId)
	if success {
		return c.JSON(fiber.Map{"message": msg, "uuid": uuid})
	}
	return c.Status(401).JSON(fiber.Map{"error": msg})
}
