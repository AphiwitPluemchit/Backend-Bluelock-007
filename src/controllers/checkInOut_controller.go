package controllers

import (
	"Backend-Bluelock-007/src/services"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func GenerateLink(c *fiber.Ctx) error {
	// เปลี่ยนจาก ActivityItemId → ActivityId
	var body struct {
		ActivityId string `json:"activityId"`
		Type       string `json:"type"`
	}

	if err := c.BodyParser(&body); err != nil || body.ActivityId == "" || body.Type == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ activityId และ type"})
	}

	uuid, err := services.GenerateCheckinUUID(body.ActivityId, body.Type)

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
func Checkout(c *fiber.Ctx) error {
	uuid := c.Params("uuid")

	var body struct {
		UserId string `json:"userId"` // จาก client ที่สแกน
	}
	if err := c.BodyParser(&body); err != nil || body.UserId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ userId"})
	}

	success, msg := services.Checkout(uuid, body.UserId) // ✅ ใช้ฟังก์ชันเดียวกัน เพราะ type ต่างกัน
	if success {
		return c.JSON(fiber.Map{"message": msg, "uuid": uuid})
	}
	return c.Status(401).JSON(fiber.Map{"error": msg})
}
func GetCheckinStatus(c *fiber.Ctx) error {
	studentId := c.Query("studentId")
	activityItemId := c.Query("activityItemId")

	if studentId == "" || activityItemId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ studentId และ activityItemId"})
	}

	status, err := services.GetCheckinStatus(studentId, activityItemId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(status)
}
