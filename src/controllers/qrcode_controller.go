package controllers

import (
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

// GenerateCheckInQRCodeHandler - สร้าง QR Code สำหรับเช็คชื่อ
func GenerateCheckInQRCodeHandler(c *fiber.Ctx) error {
	activityID := c.Params("activityId")
	if activityID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "activityId is required",
		})
	}

	qrCodePath, err := services.CreateCheckInQRCode(activityID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate check-in QR Code",
		})
	}

	return c.JSON(fiber.Map{
		"message":   "Check-in QR Code generated successfully",
		"qrCodeUrl": qrCodePath,
	})
}

// GenerateCheckOutQRCodeHandler - สร้าง QR Code สำหรับเช็คชื่อออก
func GenerateCheckOutQRCodeHandler(c *fiber.Ctx) error {
	activityID := c.Params("activityId")
	if activityID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "activityId is required",
		})
	}

	qrCodePath, err := services.CreateCheckOutQRCode(activityID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate check-out QR Code",
		})
	}

	return c.JSON(fiber.Map{
		"message":   "Check-out QR Code generated successfully",
		"qrCodeUrl": qrCodePath,
	})
}
