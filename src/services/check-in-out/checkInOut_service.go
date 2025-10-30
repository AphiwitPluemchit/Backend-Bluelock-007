package checkInOut

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/enrollments"
	hourhistory "Backend-Bluelock-007/src/services/hour-history"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetCheckinStatus returns all check-in/out records for a student and programItemId from Enrollment
func GetCheckinStatus(studentId, programItemId string) ([]map[string]interface{}, error) {
	uID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(programItemId)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	// อ่านจาก Enrollment.checkinoutRecord เท่านั้น
	var enrollment models.Enrollment
	err := DB.EnrollmentCollection.FindOne(
		context.TODO(),
		bson.M{"studentId": uID, "programItemId": aID},
	).Decode(&enrollment)
	if err != nil {
		// ไม่พบ enrollment ให้คืนค่า array ว่าง
		return []map[string]interface{}{}, nil
	}

	results := []map[string]interface{}{}
	if enrollment.CheckinoutRecord == nil {
		return results, nil
	}
	for _, r := range *enrollment.CheckinoutRecord {
		item := map[string]interface{}{}
		if r.Checkin != nil {
			item["checkin"] = *r.Checkin
		}
		if r.Checkout != nil {
			item["checkout"] = *r.Checkout
		}
		if len(item) > 0 {
			results = append(results, item)
		}
	}
	return results, nil
}

// Token Configuration
const (
	QR_TOKEN_EXPIRY    = 10  // 10 วินาที (QR Token หมดอายุเร็ว)
	CLAIM_TOKEN_EXPIRY = 600 // 10 นาที (Claim Token ให้เวลา Login)
)

// CreateQRToken creates a new QR token for an programId, valid for 10 seconds
func CreateQRToken(programId string, qrType string) (string, int64, error) {
	token := uuid.NewString()
	programObjID, err := primitive.ObjectIDFromHex(programId)
	if err != nil {
		log.Printf("❌ [CreateQRToken] Invalid programId: %s, error: %v", programId, err)
		return "", 0, err
	}
	now := time.Now().Unix()
	expiresAt := now + QR_TOKEN_EXPIRY // 10 วินาที
	qrToken := models.QRToken{
		Token:     token,
		ProgramID: programObjID,
		Type:      qrType,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	_, err = DB.QrTokenCollection.InsertOne(context.TODO(), qrToken)
	if err != nil {
		log.Printf("❌ [CreateQRToken] Failed to insert token: %v", err)
		return "", 0, err
	}
	log.Printf("✅ [CreateQRToken] Created token for programId: %s, type: %s, expires in %d seconds", programId, qrType, QR_TOKEN_EXPIRY)
	return token, expiresAt, nil
}

// ClaimQRTokenAnonymous - Claim QR Token แบบไม่ต้อง Login (เข้า link ครั้งแรก)
// ใช้เมื่อ Student scan QR หรือเข้า link ทันที (ไม่ต้องกดปุ่ม)
// Return: claimToken และข้อมูล QR
func ClaimQRTokenAnonymous(token string) (string, *models.QRToken, error) {
	ctx := context.TODO()
	now := time.Now()

	log.Printf("🔍 [ClaimQRTokenAnonymous] Token: %s", token)

	// 1️⃣ ตรวจสอบว่า QR Token ยังใช้ได้อยู่หรือไม่
	var qrToken models.QRToken
	err := DB.QrTokenCollection.FindOne(ctx, bson.M{
		"token":     token,
		"expiresAt": bson.M{"$gt": now.Unix()},
	}).Decode(&qrToken)

	if err != nil {
		log.Printf("❌ [ClaimQRTokenAnonymous] QR Token expired or invalid: %s", token)
		return "", nil, fmt.Errorf("QR Code หมดอายุ กรุณาสแกนใหม่")
	}

	log.Printf("✅ [ClaimQRTokenAnonymous] QR Token found: programId=%s, type=%s", qrToken.ProgramID.Hex(), qrToken.Type)

	// 2️⃣ สร้าง Claim Token (ยังไม่มี StudentID)
	claimToken := uuid.NewString()
	claimExpiresAt := now.Add(time.Duration(CLAIM_TOKEN_EXPIRY) * time.Second)

	claim := models.QRTokenClaim{
		ClaimToken:    claimToken,
		OriginalToken: token,
		ProgramID:     qrToken.ProgramID,
		Type:          qrToken.Type,
		StudentID:     nil, // ยังไม่ Login
		CreatedAt:     now,
		ExpiresAt:     claimExpiresAt,
		Used:          false,
	}

	_, err = DB.QrClaimCollection.InsertOne(ctx, claim)
	if err != nil {
		log.Printf("❌ [ClaimQRTokenAnonymous] Failed to create claim token: %v", err)
		return "", nil, fmt.Errorf("ไม่สามารถสร้าง Claim Token ได้: %v", err)
	}

	log.Printf("✅ [ClaimQRTokenAnonymous] Claim Token created: %s, expires at: %s", claimToken, claimExpiresAt.Format("2006-01-02 15:04:05"))

	return claimToken, &qrToken, nil
}

// ClaimQRToken - Student สแกน QR Code (อาจยัง Login หรือไม่)
// สร้าง Claim Token ที่มีอายุ 10 นาที เพื่อให้เวลา Login และ Check-in
func ClaimQRToken(token, studentId string) (*models.QRToken, error) {
	ctx := context.TODO()
	now := time.Now()

	log.Printf("🔍 [ClaimQRToken] Token: %s, StudentId: %s", token, studentId)

	// 1️⃣ ตรวจสอบว่า QR Token ยังใช้ได้อยู่หรือไม่ (ต้องไม่เกิน 10 วิ)
	var qrToken models.QRToken
	err := DB.QrTokenCollection.FindOne(ctx, bson.M{
		"token":     token,
		"expiresAt": bson.M{"$gt": now.Unix()},
	}).Decode(&qrToken)

	if err != nil {
		log.Printf("❌ [ClaimQRToken] QR Token expired or invalid: %s", token)
		return nil, fmt.Errorf("QR Code หมดอายุ กรุณาสแกนใหม่")
	}

	log.Printf("✅ [ClaimQRToken] QR Token found: programId=%s, type=%s", qrToken.ProgramID.Hex(), qrToken.Type)

	// 2️⃣ ถ้า Login แล้ว → ตรวจสอบว่าลงทะเบียนหรือไม่
	var studentObjID *primitive.ObjectID
	if studentId != "" {
		objID, err := primitive.ObjectIDFromHex(studentId)
		if err != nil {
			log.Printf("❌ [ClaimQRToken] Invalid studentId: %s", studentId)
			return nil, fmt.Errorf("รหัสนักศึกษาไม่ถูกต้อง")
		}
		studentObjID = &objID

		log.Printf("🔍 [ClaimQRToken] Checking enrollment for studentId: %s, programId: %s", studentId, qrToken.ProgramID.Hex())

		// ตรวจสอบ Enrollment
		itemIDs, found := enrollments.FindEnrolledItems(studentId, qrToken.ProgramID.Hex())
		if !found || len(itemIDs) == 0 {
			log.Printf("❌ [ClaimQRToken] Student not enrolled: %s", studentId)
			return nil, fmt.Errorf("คุณไม่ได้ลงทะเบียนกิจกรรมนี้")
		}

		log.Printf("✅ [ClaimQRToken] Student enrolled in %d items", len(itemIDs))

		// ✅ ตรวจสอบว่าเช็คชื่อวันนี้แล้วหรือยัง (สำหรับ checkin)
		if qrToken.Type == "checkin" {
			hasCheckedIn, _ := HasCheckedInToday(studentId, qrToken.ProgramID.Hex())
			if hasCheckedIn {
				log.Printf("❌ [ClaimQRToken] Already checked in today: %s", studentId)
				return nil, fmt.Errorf("คุณได้เช็คชื่อเข้าแล้วในวันนี้")
			}
			log.Printf("✅ [ClaimQRToken] Student has not checked in today")
		}
	}

	// 3️⃣ สร้าง Claim Token (หมดอายุ 10 นาที)
	claimToken := uuid.NewString()
	claimExpiresAt := now.Add(time.Duration(CLAIM_TOKEN_EXPIRY) * time.Second)

	claim := models.QRTokenClaim{
		ClaimToken:    claimToken,
		OriginalToken: token,
		ProgramID:     qrToken.ProgramID,
		Type:          qrToken.Type,
		StudentID:     studentObjID,
		CreatedAt:     now,
		ExpiresAt:     claimExpiresAt,
		Used:          false,
	}

	_, err = DB.QrClaimCollection.InsertOne(ctx, claim)
	if err != nil {
		log.Printf("❌ [ClaimQRToken] Failed to create claim token: %v", err)
		return nil, fmt.Errorf("ไม่สามารถสร้าง Claim Token ได้: %v", err)
	}

	log.Printf("✅ [ClaimQRToken] Claim Token created: %s, expires at: %s", claimToken, claimExpiresAt.Format("2006-01-02 15:04:05"))

	// 4️⃣ Return QR Token info พร้อม Claim Token
	qrToken.ClaimedByStudentID = studentObjID
	return &qrToken, nil
}

// HasCheckedInToday - ตรวจสอบว่าเช็คชื่อเข้าวันนี้แล้วหรือยัง
func HasCheckedInToday(studentId, programId string) (bool, error) {
	ctx := context.TODO()
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return false, err
	}

	programObjID, err := primitive.ObjectIDFromHex(programId)
	if err != nil {
		return false, err
	}

	loc, _ := time.LoadLocation("Asia/Bangkok")
	dateKey := time.Now().In(loc).Format("2006-01-02")

	fmt.Printf("🔍 [HasCheckedInToday] Checking check-in for studentId: %s, programId: %s on %s", studentId, programId, dateKey)

	// หา Enrollment
	var enrollment models.Enrollment
	err = DB.EnrollmentCollection.FindOne(ctx, bson.M{
		"studentId": studentObjID,
		"programId": programObjID,
	}).Decode(&enrollment)

	if err != nil {
		return false, nil
	}

	// ตรวจสอบ Checkin Record วันนี้
	if enrollment.CheckinoutRecord != nil {
		for _, record := range *enrollment.CheckinoutRecord {
			if record.Checkin != nil {
				recDate := record.Checkin.In(loc).Format("2006-01-02")
				if recDate == dateKey {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// HasCheckedOutToday - ตรวจสอบว่าเช็คชื่อออกวันนี้แล้วหรือยัง
func HasCheckedOutToday(studentId, programId string) (bool, error) {
	ctx := context.TODO()
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return false, err
	}

	programObjID, err := primitive.ObjectIDFromHex(programId)
	if err != nil {
		return false, err
	}

	loc, _ := time.LoadLocation("Asia/Bangkok")
	dateKey := time.Now().In(loc).Format("2006-01-02")

	// หา Enrollment
	var enrollment models.Enrollment
	err = DB.EnrollmentCollection.FindOne(ctx, bson.M{
		"studentId": studentObjID,
		"programId": programObjID,
	}).Decode(&enrollment)

	if err != nil {
		return false, nil
	}

	// ตรวจสอบ Checkout Record วันนี้
	if enrollment.CheckinoutRecord != nil {
		for _, record := range *enrollment.CheckinoutRecord {
			if record.Checkout != nil {
				recDate := record.Checkout.In(loc).Format("2006-01-02")
				if recDate == dateKey {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// ValidateQRToken checks if the token is valid for the student (claimed and not expired)
// Legacy function - ใช้กับระบบเก่า
func ValidateQRToken(token, studentId string) (*models.QRToken, error) {
	ctx := context.TODO()
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return nil, err
	}
	var claim struct {
		Token     string             `bson:"token"`
		StudentID primitive.ObjectID `bson:"studentId"`
		ProgramID primitive.ObjectID `bson:"programId"`
		Type      string             `bson:"type"`
		ClaimedAt time.Time          `bson:"claimedAt"`
		ExpireAt  time.Time          `bson:"expireAt"`
	}
	err = DB.QrClaimCollection.FindOne(ctx, bson.M{"token": token, "studentId": studentObjID, "expireAt": bson.M{"$gt": time.Now()}}).Decode(&claim)
	if err != nil {
		return nil, fmt.Errorf("QR token not claimed or expired")
	}

	// ตรวจสอบว่านักศึกษาได้ลงทะเบียนในกิจกรรมนี้หรือไม่
	itemIDs, found := enrollments.FindEnrolledItems(studentId, claim.ProgramID.Hex())
	if !found || len(itemIDs) == 0 {
		return nil, fmt.Errorf("คุณไม่ได้ลงทะเบียนกิจกรรมนี้")
	}

	return &models.QRToken{
		Token:              claim.Token,
		ProgramID:          claim.ProgramID,
		Type:               claim.Type,
		ClaimedByStudentID: &studentObjID,
	}, nil
}

// ValidateClaimToken - ตรวจสอบ Claim Token (หลัง Login)
// ใช้เมื่อ Student กลับมาหลังจาก Login แล้ว
func ValidateClaimToken(claimToken, studentId string) (*models.QRTokenClaim, error) {
	ctx := context.TODO()
	now := time.Now()

	log.Printf("🔍 [ValidateClaimToken] ClaimToken: %s, StudentId: %s", claimToken, studentId)

	// 1️⃣ หา Claim Token
	var claim models.QRTokenClaim
	err := DB.QrClaimCollection.FindOne(ctx, bson.M{
		"claimToken": claimToken,
		"expiresAt":  bson.M{"$gt": now},
		"used":       false,
	}).Decode(&claim)

	if err != nil {
		log.Printf("❌ [ValidateClaimToken] Claim Token expired or not found: %s", claimToken)
		return nil, fmt.Errorf("session หมดอายุ กรุณาสแกน QR ใหม่")
	}

	log.Printf("✅ [ValidateClaimToken] Claim Token found: programId=%s, type=%s", claim.ProgramID.Hex(), claim.Type)

	// 2️⃣ ถ้ายังไม่มี StudentID → อัปเดต (กรณี Scan ก่อน Login)
	if claim.StudentID == nil && studentId != "" {
		log.Printf("🔄 [ValidateClaimToken] Updating claim token with studentId: %s", studentId)

		studentObjID, err := primitive.ObjectIDFromHex(studentId)
		if err != nil {
			log.Printf("❌ [ValidateClaimToken] Invalid studentId: %s", studentId)
			return nil, fmt.Errorf("รหัสนักศึกษาไม่ถูกต้อง")
		}

		// ตรวจสอบ Enrollment
		itemIDs, found := enrollments.FindEnrolledItems(studentId, claim.ProgramID.Hex())
		if !found || len(itemIDs) == 0 {
			log.Printf("❌ [ValidateClaimToken] Student not enrolled: %s", studentId)
			return nil, fmt.Errorf("คุณไม่ได้ลงทะเบียนกิจกรรมนี้")
		}

		log.Printf("✅ [ValidateClaimToken] Student enrolled in %d items", len(itemIDs))

		// อัปเดต StudentID
		_, err = DB.QrClaimCollection.UpdateOne(ctx, bson.M{
			"claimToken": claimToken,
		}, bson.M{
			"$set": bson.M{"studentId": studentObjID},
		})

		if err != nil {
			log.Printf("❌ [ValidateClaimToken] Failed to update claim token: %v", err)
			return nil, fmt.Errorf("ไม่สามารถอัปเดตข้อมูลได้")
		}

		log.Printf("✅ [ValidateClaimToken] Claim token updated with studentId")
		claim.StudentID = &studentObjID
	}

	// 3️⃣ ถ้ามี StudentID แล้ว → ตรวจสอบว่าตรงกันหรือไม่
	if claim.StudentID != nil && studentId != "" {
		studentObjID, _ := primitive.ObjectIDFromHex(studentId)
		if claim.StudentID.Hex() != studentObjID.Hex() {
			log.Printf("❌ [ValidateClaimToken] Claim token belongs to different student: %s vs %s", claim.StudentID.Hex(), studentObjID.Hex())
			return nil, fmt.Errorf("claim Token นี้ไม่ได้เป็นของคุณ")
		}
	}

	log.Printf("✅ [ValidateClaimToken] Validation successful")
	return &claim, nil
}

// MarkClaimTokenAsUsed - ทำเครื่องหมาย Claim Token ว่าใช้แล้ว
func MarkClaimTokenAsUsed(claimToken string) error {
	ctx := context.TODO()
	log.Printf("🔒 [MarkClaimTokenAsUsed] Marking claim token as used: %s", claimToken)

	_, err := DB.QrClaimCollection.UpdateOne(ctx, bson.M{
		"claimToken": claimToken,
	}, bson.M{
		"$set": bson.M{"used": true},
	})

	if err != nil {
		log.Printf("❌ [MarkClaimTokenAsUsed] Failed to mark as used: %v", err)
		return err
	}

	log.Printf("✅ [MarkClaimTokenAsUsed] Claim token marked as used")
	return nil
}

// GetProgramFormId ดึง formId จาก programId
func GetProgramFormId(programId string) (string, error) {
	ctx := context.TODO()
	programObjID, err := primitive.ObjectIDFromHex(programId)
	if err != nil {
		return "", fmt.Errorf("invalid program ID format")
	}

	var program struct {
		FormID primitive.ObjectID `bson:"formId"`
	}

	err = DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programObjID}).Decode(&program)
	if err != nil {
		return "", fmt.Errorf("program not found")
	}

	// ตรวจสอบว่า formId เป็น zero value หรือไม่
	if program.FormID.IsZero() {
		return "", fmt.Errorf("program does not have a form")
	}

	return program.FormID.Hex(), nil
}

type AddHoursForStudentResult struct {
	ProgramItemID string                     `json:"programItemId"`
	ProgramName   string                     `json:"programName"`
	SkillType     string                     `json:"skillType"`
	TotalStudents int                        `json:"totalStudents"`
	SuccessCount  int                        `json:"successCount"`
	ErrorCount    int                        `json:"errorCount"`
	Results       []models.HourChangeHistory `json:"results"`
}

func AddHoursForStudent(programItemId string) (*AddHoursForStudentResult, error) {
	ctx := context.TODO()
	programItemObjID, err := primitive.ObjectIDFromHex(programItemId)
	if err != nil {
		return nil, fmt.Errorf("invalid programItemId format: %v", err)
	}

	// 1) ProgramItem
	var programItem models.ProgramItem
	if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": programItemObjID}).Decode(&programItem); err != nil {
		return nil, fmt.Errorf("program item not found: %v", err)
	}
	if programItem.Hour == nil {
		return nil, fmt.Errorf("program item has no hour value")
	}

	// 2) Program
	var program models.Program
	if err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programItem.ProgramID}).Decode(&program); err != nil {
		return nil, fmt.Errorf("program not found: %v", err)
	}

	// 3) Enrollments
	cur, err := DB.EnrollmentCollection.Find(ctx, bson.M{"programItemId": programItemObjID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch enrollments: %v", err)
	}
	defer cur.Close(ctx)

	var enrollments []models.Enrollment
	if err := cur.All(ctx, &enrollments); err != nil {
		return nil, fmt.Errorf("failed to decode enrollments: %v", err)
	}

	// 4) Result
	result := &AddHoursForStudentResult{
		ProgramItemID: programItemId,
		ProgramName:   deref(program.Name), // เผื่อ program.Name เป็น *string
		SkillType:     program.Skill,
		TotalStudents: len(enrollments),
		Results:       make([]models.HourChangeHistory, 0, len(enrollments)),
	}

	// 5) Process each enrollment
	for _, en := range enrollments {
		_, err := processStudentHours(
			ctx,
			en.ID, // ส่ง enrollmentId เข้าไป
			en.StudentID,
			programItemObjID,
			programItem,
			program.Skill,
		)
		if err != nil {
			result.ErrorCount++
			// กรณี error: แนบข้อมูลเท่าที่รู้ (ไว้โชว์ใน response)
			programName := deref(program.Name) // ใช้ชื่อ program จริง
			if programName == "" {
				programName = "Unknown Program"
			}
			result.Results = append(result.Results, models.HourChangeHistory{
				ID:           primitive.NewObjectID(),
				StudentID:    en.StudentID,
				EnrollmentID: &en.ID,
				SourceType:   "program",
				SourceID:     &programItem.ProgramID,
				SkillType:    program.Skill,
				HourChange:   0,
				Title:        programName, // ใช้ชื่อ program แทน
				Remark:       fmt.Sprintf("Error: %v", err),
				ChangeAt:     time.Now(),
			})
			continue
		}

		result.SuccessCount++
		// h is now nil, no need to append it
		// result.Results = append(result.Results, *h)
	}

	return result, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// findTodayCheckinRecord หา record ของวันที่ระบุที่มี check-in อยู่แล้ว
// คืนค่า index ของ record ที่เจอ หรือ -1 ถ้าไม่เจอ
func findTodayCheckinRecord(records []models.CheckinoutRecord, dateKey string, loc *time.Location) int {
	for i := range records {
		if records[i].Checkin != nil {
			recDate := records[i].Checkin.In(loc).Format("2006-01-02")
			if recDate == dateKey {
				return i
			}
		}
	}
	return -1
}

// SaveCheckInOut บันทึกการเช็คชื่อเข้า/ออก
func SaveCheckInOut(studentId, programId, checkType string) error {
	ctx := context.TODO()

	log.Printf("📝 [SaveCheckInOut] StudentId: %s, ProgramId: %s, Type: %s", studentId, programId, checkType)

	// หา programItemId ที่นิสิตลงทะเบียนใน program นี้ (1 enrollment ต่อ 1 program)
	programItemId, found := enrollments.FindEnrolledProgramItem(studentId, programId)
	if !found {
		log.Printf("❌ [SaveCheckInOut] Student not enrolled: %s", studentId)
		return fmt.Errorf("คุณไม่ได้ลงทะเบียนกิจกรรมนี้")
	}

	log.Printf("✅ [SaveCheckInOut] Found program item: %s", programItemId)

	uID, err1 := primitive.ObjectIDFromHex(studentId)
	programItemID, err2 := primitive.ObjectIDFromHex(programItemId)
	if err1 != nil || err2 != nil {
		log.Printf("❌ [SaveCheckInOut] Invalid ID format")
		return fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Bangkok")
	dateKey := now.In(loc).Format("2006-01-02")

	// 1) ดึงข้อมูล Enrollment & ProgramItem
	var enrollment models.Enrollment
	if err := DB.EnrollmentCollection.FindOne(ctx,
		bson.M{"studentId": uID, "programItemId": programItemID},
	).Decode(&enrollment); err != nil {
		return fmt.Errorf("ไม่พบการลงทะเบียนของกิจกรรมนี้")
	}

	var programItem models.ProgramItem
	if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": programItemID}).Decode(&programItem); err != nil {
		return fmt.Errorf("ไม่พบข้อมูล program item")
	}

	// 2) ตรวจสอบว่าวันนี้อยู่ในตารางกิจกรรมหรือไม่
	today := now.In(loc).Format("2006-01-02")
	allowed := false
	for _, d := range programItem.Dates {
		if d.Date == today {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("ไม่อนุญาตเช็คชื่อ: วันนี้ (%s) ไม่มีตารางกิจกรรมของรายการนี้", today)
	}

	// 3) เตรียม records
	records := []models.CheckinoutRecord{}
	if enrollment.CheckinoutRecord != nil {
		records = *enrollment.CheckinoutRecord
	}

	// 4) บันทึก Check-in หรือ Check-out
	switch checkType {
	case "checkin":
		log.Printf("🔍 [SaveCheckInOut] Processing check-in for date: %s", dateKey)

		// ตรวจสอบว่าเคยเช็คอินวันนี้แล้วหรือไม่
		if idx := findTodayCheckinRecord(records, dateKey, loc); idx >= 0 {
			log.Printf("❌ [SaveCheckInOut] Already checked in today")
			return fmt.Errorf("คุณได้เช็คชื่อ checkin แล้วในวันนี้")
		}

		// สร้าง record ใหม่สำหรับ check-in วันนี้
		t := now
		records = append(records, models.CheckinoutRecord{
			ID:      primitive.NewObjectID(),
			Checkin: &t,
		})

		log.Printf("✅ [SaveCheckInOut] Check-in record created")

		// อัปเดต Hour Change History status จาก Upcoming → Participating
		if err := hourhistory.RecordCheckinActivity(ctx, enrollment.ID, dateKey); err != nil {
			log.Printf("⚠️  [SaveCheckInOut] Warning: failed to record checkin activity: %v", err)
		} else {
			log.Printf("✅ [SaveCheckInOut] Hour history updated")
		}

	case "checkout":
		log.Printf("🔍 [SaveCheckInOut] Processing check-out for date: %s", dateKey)

		// หา record ของวันนี้ที่มี check-in อยู่แล้ว
		idx := findTodayCheckinRecord(records, dateKey, loc)

		if idx >= 0 {
			// เจอ record ของวันนี้
			if records[idx].Checkout != nil {
				log.Printf("❌ [SaveCheckInOut] Already checked out today")
				return fmt.Errorf("คุณได้เช็คชื่อ checkout แล้วในวันนี้")
			}
			// อัปเดต checkout
			t := now
			records[idx].Checkout = &t
			log.Printf("✅ [SaveCheckInOut] Check-out updated on existing record")
		} else {
			// ไม่เจอ record ของวันนี้ → สร้างใหม่ (checkout-only case)
			t := now
			records = append(records, models.CheckinoutRecord{
				ID:       primitive.NewObjectID(),
				Checkout: &t,
			})
			log.Printf("✅ [SaveCheckInOut] Check-out record created (checkout-only)")
		}

	default:
		log.Printf("❌ [SaveCheckInOut] Invalid check type: %s", checkType)
		return fmt.Errorf("ประเภทการเช็คชื่อไม่ถูกต้อง")
	}

	// 5) คำนวณ attendedAllDays (เช็คว่ามี checkin/checkout ครบทุกวันหรือไม่)
	attendedAll := checkAttendedAllDays(records, programItem.Dates)
	log.Printf("📊 [SaveCheckInOut] Attended all days: %v", attendedAll)

	// 6) บันทึกลง Enrollment
	update := bson.M{
		"$set": bson.M{
			"checkinoutRecord": records,
			"attendedAllDays":  attendedAll,
		},
	}
	if _, err := DB.EnrollmentCollection.UpdateOne(
		ctx,
		bson.M{"studentId": uID, "programItemId": programItemID},
		update,
	); err != nil {
		log.Printf("❌ [SaveCheckInOut] Failed to update enrollment: %v", err)
		return err
	}

	log.Printf("✅ [SaveCheckInOut] %s successful for student: %s", checkType, studentId)
	return nil
}

// checkAttendedAllDays ตรวจสอบว่านิสิตเข้าร่วมครบทุกวันหรือไม่
// เช็คว่ามี checkin และ checkout ครบทุกวันตาม programItem.Dates
func checkAttendedAllDays(records []models.CheckinoutRecord, dates []models.Dates) bool {
	loc, _ := time.LoadLocation("Asia/Bangkok")

	// สร้าง map ของ records ตามวันที่
	recordsByDate := make(map[string]models.CheckinoutRecord)
	for _, r := range records {
		var dateKey string
		if r.Checkin != nil {
			dateKey = r.Checkin.In(loc).Format("2006-01-02")
		} else if r.Checkout != nil {
			dateKey = r.Checkout.In(loc).Format("2006-01-02")
		}
		if dateKey != "" {
			recordsByDate[dateKey] = r
		}
	}

	// ตรวจสอบทุกวันในตาราง - ต้องมีทั้ง checkin และ checkout
	for _, d := range dates {
		record, exists := recordsByDate[d.Date]
		if !exists || record.Checkin == nil || record.Checkout == nil {
			return false
		}
	}

	return true
}
