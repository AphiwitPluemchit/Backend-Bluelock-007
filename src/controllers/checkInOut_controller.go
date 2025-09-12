package controllers

import (
	checkInOut "Backend-Bluelock-007/src/services/check-in-out"
	"Backend-Bluelock-007/src/services/enrollments"

	"github.com/gofiber/fiber/v2"
)

func GetCheckinStatus(c *fiber.Ctx) error {
	studentId := c.Query("studentId")
	programId := c.Query("programId")

	if studentId == "" || programId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ studentId และ programId"})
	}

	// ดึง programItemIds ทั้งหมดที่นิสิตลงทะเบียนไว้ใน programId นี้
	programItemIds, found := enrollments.FindEnrolledItems(studentId, programId)
	if !found || len(programItemIds) == 0 {
		return c.JSON([]interface{}{}) // ส่ง array ว่าง
	}

	// ใช้แค่ programItemId อันแรก
	results, err := checkInOut.GetCheckinStatus(studentId, programItemIds[0])
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(results)
}

// POST /admin/qr-token
func AdminCreateQRToken(c *fiber.Ctx) error {
	var body struct {
		ProgramId string `json:"programId"`
		Type      string `json:"type"`
	}
	if err := c.BodyParser(&body); err != nil || body.ProgramId == "" || body.Type == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ programId และ type"})
	}
	token, expiresAt, err := checkInOut.CreateQRToken(body.ProgramId, body.Type)
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
	qrToken, err := checkInOut.ClaimQRToken(token, studentId)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"programId": qrToken.ProgramID.Hex(), "token": qrToken.Token, "type": qrToken.Type})
}

// GET /Student/validate/:token
func StudentValidateQRToken(c *fiber.Ctx) error {
	userIdRaw := c.Locals("userId")
	studentId, ok := userIdRaw.(string)
	if !ok || studentId == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}
	token := c.Params("token")
	qrToken, err := checkInOut.ValidateQRToken(token, studentId)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"programId": qrToken.ProgramID.Hex(),
		"token":     qrToken.Token,
		"type":      qrToken.Type,
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
	qrToken, err := checkInOut.ClaimQRToken(body.Token, studentId)
	if err != nil && (err.Error() == "QR token expired or invalid" || err.Error() == "QR Code นี้หมดอายุแล้ว หรือ ยังไม่ถูก claim") {
		// fallback validate
		qrToken, err = checkInOut.ValidateQRToken(body.Token, studentId)
	}
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	err = checkInOut.RecordCheckin(studentId, qrToken.ProgramID.Hex(), "checkin")
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
	qrToken, err := checkInOut.ClaimQRToken(body.Token, studentId)
	if err != nil && (err.Error() == "QR token expired or invalid" || err.Error() == "QR token not claimed or expired") {
		// fallback validate
		qrToken, err = checkInOut.ValidateQRToken(body.Token, studentId)
	}
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	err = checkInOut.RecordCheckin(studentId, qrToken.ProgramID.Hex(), "checkout")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "ลงทะเบียนออกสำเร็จ"})
}

// GET /Student/program/:programId/form
func GetProgramForm(c *fiber.Ctx) error {
	programId := c.Params("programId")
	if programId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ programId"})
	}

	formId, err := checkInOut.GetProgramFormId(programId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"formId": formId})
}
func AddHoursForStudent(c *fiber.Ctx) error {
	programItemId := c.Params("programItemId")
	if programItemId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ programItemId"})
	}

	result, err := checkInOut.AddHoursForStudent(programItemId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}
	// ดูผลลัพธ์
	return c.JSON(fiber.Map{"message": "เพิ่มชั่วโมงให้นิสิตสำเร็จ", "data": result})
}
