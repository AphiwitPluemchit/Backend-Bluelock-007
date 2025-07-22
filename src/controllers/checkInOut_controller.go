package controllers

import (
	"Backend-Bluelock-007/src/services"
	"Backend-Bluelock-007/src/services/enrollments"

	"github.com/gofiber/fiber/v2"
)

// func GenerateLink(c *fiber.Ctx) error {
// 	// เปลี่ยนจาก ActivityItemId → ActivityId
// 	var body struct {
// 		ActivityId string `json:"activityId"`
// 		Type       string `json:"type"`
// 	}

// 	if err := c.BodyParser(&body); err != nil || body.ActivityId == "" || body.Type == "" {
// 		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ activityId และ type"})
// 	}

// 	uuid, err := services.GenerateCheckinUUID(body.ActivityId, body.Type)

// 	if err != nil {
// 		return c.Status(500).JSON(fiber.Map{"error": "ไม่สามารถสร้าง UUID ได้"})
// 	}

//		return c.JSON(fiber.Map{
//			"uuid": uuid,
//			"url":  fmt.Sprintf("/%s/%s", body.Type, uuid),
//		})
//	}

// func Checkin(c *fiber.Ctx) error {
// 	uuid := c.Params("uuid")

// 	var body struct {
// 		UserId string `json:"userId"` // ✅ รับจาก frontend
// 	}
// 	if err := c.BodyParser(&body); err != nil || body.UserId == "" {
// 		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ userId"})
// 	}

// 	success, msg := services.Checkin(uuid, body.UserId)
// 	if success {
// 		return c.JSON(fiber.Map{"message": msg, "uuid": uuid})
// 	}
// 	return c.Status(401).JSON(fiber.Map{"error": msg})
// }

// func Checkout(c *fiber.Ctx) error {
// 	uuid := c.Params("uuid")

// 	var body struct {
// 		UserId       string `json:"userId"`
// 		EvaluationId string `json:"evaluationId"`
// 	}
// 	if err := c.BodyParser(&body); err != nil || body.UserId == "" || body.EvaluationId == "" {
// 		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ userId และ evaluationId"})
// 	}

// 	success, msg := services.Checkout(uuid, body.UserId, body.EvaluationId)
// 	if success {
// 		return c.JSON(fiber.Map{"message": msg, "uuid": uuid})
// 	}
// 	return c.Status(401).JSON(fiber.Map{"error": msg})
// }

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
	url := "/student/qr/" + token
	return c.JSON(fiber.Map{"token": token, "expiresAt": expiresAt, "url": url, "type": body.Type})
}

// GET /student/qr/:token
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

// POST /student/checkin
func StudentCheckin(c *fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	studentId := c.Locals("userId").(string)
	if err := c.BodyParser(&body); err != nil || body.Token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ token"})
	}
	qrToken, err := services.ValidateQRToken(body.Token, studentId)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	err = services.RecordCheckin(studentId, qrToken.ActivityID.Hex(), "checkin")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "checkin success"})
}

// POST /student/checkout
func StudentCheckout(c *fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	studentId := c.Locals("userId").(string)
	if err := c.BodyParser(&body); err != nil || body.Token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ token"})
	}
	qrToken, err := services.ValidateQRToken(body.Token, studentId)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	err = services.RecordCheckin(studentId, qrToken.ActivityID.Hex(), "checkout")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "checkout success"})
}
