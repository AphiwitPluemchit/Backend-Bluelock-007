package controllers

import (
	"Backend-Bluelock-007/src/services"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func GenerateLink(c *fiber.Ctx) error {
	var body struct {
		ActivityItemId string `json:"activityItemId"`
		Type           string `json:"type"`
	}

	if err := c.BodyParser(&body); err != nil || body.ActivityItemId == "" || body.Type == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ activityItemId และ type"})
	}

	uuid, err := services.GenerateCheckinUUID(body.ActivityItemId, body.Type)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "ไม่สามารถสร้าง UUID ได้"})
	}

	return c.JSON(fiber.Map{
		"uuid": uuid,
		"url":  fmt.Sprintf("/%s/%s", body.Type, uuid),
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
