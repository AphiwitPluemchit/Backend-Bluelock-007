package controllers

import (
	checkInOut "Backend-Bluelock-007/src/services/check-in-out"
	"Backend-Bluelock-007/src/services/enrollments"

	"github.com/gofiber/fiber/v2"
)

// func ClearToken(c *fiber.Ctx) error {
// 	programId := c.Params("programId")
// 	if programId == "" {
// 		return c.Status(400).JSON(fiber.Map{"error": "programId is required1"})
// 	}
// 	// Convert programId string to MongoDB ObjectID
// 	objectId, err := primitive.ObjectIDFromHex(programId)
// 	if err != nil {
// 		return c.Status(400).JSON(fiber.Map{"error": "Invalid programId"})
// 	}
// 	err = checkInOut.ClearToken(objectId)
// 	if err != nil {
// 		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
// 	}
// 	return c.Status(200).JSON(fiber.Map{"message": "Token cleared successfully"})
// }

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

// GET /public/qr/:token - Anonymous claim (ไม่ต้อง Login)
func PublicClaimQRToken(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ token"})
	}

	claimToken, qrToken, err := checkInOut.ClaimQRTokenAnonymous(token)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"claimToken": claimToken,
		"programId":  qrToken.ProgramID.Hex(),
		"type":       qrToken.Type,
		"message":    "Claim สำเร็จ กรุณา Login เพื่อเช็คชื่อ",
	})
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

// GET /Student/validate/:token (Legacy)
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

// GET /Student/validate-claim/:claimToken
func StudentValidateClaimToken(c *fiber.Ctx) error {
	userIdRaw := c.Locals("userId")
	studentId, ok := userIdRaw.(string)
	if !ok || studentId == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}
	claimToken := c.Params("claimToken")
	claim, err := checkInOut.ValidateClaimToken(claimToken, studentId)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"claimToken": claim.ClaimToken,
		"programId":  claim.ProgramID.Hex(),
		"type":       claim.Type,
		"expiresAt":  claim.ExpiresAt.Unix(),
	})
}

// POST /Student/checkin
func StudentCheckin(c *fiber.Ctx) error {
	var body struct {
		Token      string `json:"token"`      // QR Token หรือ Claim Token
		ClaimToken string `json:"claimToken"` // Claim Token (ถ้ามี)
	}
	studentId := c.Locals("userId").(string)
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ข้อมูลไม่ถูกต้อง"})
	}

	var programId string
	var checkErr error

	// 1️⃣ ถ้ามี ClaimToken → ตรวจสอบก่อนว่า check-in แล้วหรือยัง

	// ✅ เช็คว่า check-in ไปแล้วหรือยัง (ก่อนบันทึก)
	hasCheckedIn, _ := checkInOut.HasCheckedInToday(studentId, programId)
	if hasCheckedIn {
		return c.Status(400).JSON(fiber.Map{"error": "คุณได้เช็คชื่อเข้าแล้วในวันนี้"})
	}

	if body.ClaimToken != "" {
		claim, err := checkInOut.ValidateClaimToken(body.ClaimToken, studentId)
		if err != nil {
			return c.Status(403).JSON(fiber.Map{"error": err.Error()})
		}
		programId = claim.ProgramID.Hex()

	} else if body.Token != "" {
		// 2️⃣ ถ้าไม่มี ClaimToken → ใช้ Token เดิม (Legacy)
		qrToken, err := checkInOut.ClaimQRToken(body.Token, studentId)
		if err != nil && (err.Error() == "QR token expired or invalid" || err.Error() == "QR Code หมดอายุ กรุณาสแกนใหม่") {
			// fallback validate (legacy)
			qrToken, err = checkInOut.ValidateQRToken(body.Token, studentId)
		}
		if err != nil {
			return c.Status(403).JSON(fiber.Map{"error": err.Error()})
		}
		programId = qrToken.ProgramID.Hex()
	} else {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ token หรือ claimToken"})
	}

	// 3️⃣ บันทึก Check-in
	checkErr = checkInOut.SaveCheckInOut(studentId, programId, "checkin")
	if checkErr != nil {
		return c.Status(400).JSON(fiber.Map{"error": checkErr.Error()})
	}

	// 4️⃣ ทำเครื่องหมาย Claim Token ว่าใช้แล้ว (ถ้ามี)
	if body.ClaimToken != "" {
		checkInOut.MarkClaimTokenAsUsed(body.ClaimToken)
	}

	return c.JSON(fiber.Map{"message": "ลงทะเบียนเข้าสำเร็จ"})
}

// POST /Student/checkout
func StudentCheckout(c *fiber.Ctx) error {
	var body struct {
		Token      string `json:"token"`      // QR Token หรือ Claim Token
		ClaimToken string `json:"claimToken"` // Claim Token (ถ้ามี)
	}
	studentId := c.Locals("userId").(string)
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ข้อมูลไม่ถูกต้อง"})
	}

	var programId string
	var checkErr error

	// 1️⃣ ถ้ามี ClaimToken → ตรวจสอบก่อนว่า check-out แล้วหรือยัง
	if body.ClaimToken != "" {
		claim, err := checkInOut.ValidateClaimToken(body.ClaimToken, studentId)
		if err != nil {
			return c.Status(403).JSON(fiber.Map{"error": err.Error()})
		}
		programId = claim.ProgramID.Hex()

		// ✅ เช็คว่า check-out ไปแล้วหรือยัง (ก่อนบันทึก)
		hasCheckedOut, _ := checkInOut.HasCheckedOutToday(studentId, programId)
		if hasCheckedOut {
			return c.Status(400).JSON(fiber.Map{"error": "คุณได้เช็คชื่อออกแล้วในวันนี้"})
		}
	} else if body.Token != "" {
		// 2️⃣ ถ้าไม่มี ClaimToken → ใช้ Token เดิม (Legacy)
		qrToken, err := checkInOut.ClaimQRToken(body.Token, studentId)
		if err != nil && (err.Error() == "QR token expired or invalid" || err.Error() == "QR token not claimed or expired" || err.Error() == "QR Code หมดอายุ กรุณาสแกนใหม่") {
			// fallback validate (legacy)
			qrToken, err = checkInOut.ValidateQRToken(body.Token, studentId)
		}
		if err != nil {
			return c.Status(403).JSON(fiber.Map{"error": err.Error()})
		}
		programId = qrToken.ProgramID.Hex()
	} else {
		return c.Status(400).JSON(fiber.Map{"error": "ต้องระบุ token หรือ claimToken"})
	}

	// 3️⃣ บันทึก Check-out
	checkErr = checkInOut.SaveCheckInOut(studentId, programId, "checkout")
	if checkErr != nil {
		return c.Status(400).JSON(fiber.Map{"error": checkErr.Error()})
	}

	// 4️⃣ ทำเครื่องหมาย Claim Token ว่าใช้แล้ว (ถ้ามี)
	if body.ClaimToken != "" {
		checkInOut.MarkClaimTokenAsUsed(body.ClaimToken)
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
