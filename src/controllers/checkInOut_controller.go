package controllers

import (
	services "Backend-Bluelock-007/src/services/check-in-out"
	"Backend-Bluelock-007/src/services/enrollments"

	"github.com/gofiber/fiber/v2"
)

func GetCheckinStatus(c *fiber.Ctx) error {
	studentId := c.Query("studentId")
	activityId := c.Query("activityId")

	if studentId == "" || activityId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ studentId และ activityId"})
	}

	// ดึง activityItemIds ทั้งหมดที่นิสิตลงทะเบียนไว้ใน activityId นี้
	activityItemIds, found := enrollments.FindEnrolledItems(studentId, activityId)
	if !found || len(activityItemIds) == 0 {
		return c.JSON([]interface{}{}) // ส่ง array ว่าง
	}

	// ใช้แค่ activityItemId อันแรก
	results, err := services.GetCheckinStatus(studentId, activityItemIds[0])
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(results)
}

// POST /admin/qr-token
func AdminCreateQRToken(c *fiber.Ctx) error {
	var body struct {
		ActivityId string `json:"activityId"`
		Type       string `json:"type"`
	}
	if err := c.BodyParser(&body); err != nil || body.ActivityId == "" || body.Type == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ activityId และ type"})
	}
	token, expiresAt, err := services.CreateQRToken(body.ActivityId, body.Type)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	url := "/Student/qr/" + token
	return c.JSON(fiber.Map{"token": token, "expiresAt": expiresAt, "url": url, "type": body.Type})
}

// GET /Student/qr/:token
func StudentClaimQRToken(c *fiber.Ctx) error {
	userIdRaw := c.Locals("userId")
	studentId, ok := userIdRaw.(string)
	if !ok || studentId == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}
	token := c.Params("token")
	qrToken, err := services.ClaimQRToken(token, studentId)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"activityId": qrToken.ActivityID.Hex(), "token": qrToken.Token, "type": qrToken.Type})
}

// GET /Student/validate/:token
func StudentValidateQRToken(c *fiber.Ctx) error {
	userIdRaw := c.Locals("userId")
	studentId, ok := userIdRaw.(string)
	if !ok || studentId == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}
	token := c.Params("token")
	qrToken, err := services.ValidateQRToken(token, studentId)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"activityId": qrToken.ActivityID.Hex(),
		"token":      qrToken.Token,
		"type":       qrToken.Type,
	})
}

// POST /Student/checkin
func StudentCheckin(c *fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	studentId := c.Locals("userId").(string)
	if err := c.BodyParser(&body); err != nil || body.Token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ token"})
	}
	qrToken, err := services.ClaimQRToken(body.Token, studentId)
	if err != nil && (err.Error() == "QR token expired or invalid" || err.Error() == "QR Code นี้หมดอายุแล้ว หรือ ยังไม่ถูก claim") {
		// fallback validate
		qrToken, err = services.ValidateQRToken(body.Token, studentId)
	}
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	err = services.RecordCheckin(studentId, qrToken.ActivityID.Hex(), "checkin")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "ลงทะเบียนเข้าสำเร็จ"})
}

// POST /Student/checkout
func StudentCheckout(c *fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	studentId := c.Locals("userId").(string)
	if err := c.BodyParser(&body); err != nil || body.Token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ token"})
	}
	qrToken, err := services.ClaimQRToken(body.Token, studentId)
	if err != nil && (err.Error() == "QR token expired or invalid" || err.Error() == "QR token not claimed or expired") {
		// fallback validate
		qrToken, err = services.ValidateQRToken(body.Token, studentId)
	}
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	err = services.RecordCheckin(studentId, qrToken.ActivityID.Hex(), "checkout")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "ลงทะเบียนออกสำเร็จ"})
}
