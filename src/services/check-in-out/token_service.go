package checkInOut

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/enrollments"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Token Configuration
const (
	QR_TOKEN_EXPIRY    = 10  // 10 วินาที (QR Token หมดอายุเร็ว)
	CLAIM_TOKEN_EXPIRY = 600 // 10 นาที (Claim Token ให้เวลา Login)
)

// ============================================
// QR Token Management (อายุ 10 วินาที)
// ============================================

// CreateQRToken สร้าง QR Token สำหรับ Admin
func CreateQRToken(programId string, qrType string) (string, int64, error) {
	programObjID, err := convertToObjectID(programId)
	if err != nil {
		log.Printf("❌ [CreateQRToken] Invalid programId: %s", programId)
		return "", 0, err
	}

	token := uuid.NewString()
	now := time.Now().Unix()
	expiresAt := now + QR_TOKEN_EXPIRY

	qrToken := models.QRToken{
		Token:     token,
		ProgramID: programObjID,
		Type:      qrType,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	_, err = DB.QrTokenCollection.InsertOne(context.TODO(), qrToken)
	if err != nil {
		log.Printf("❌ [CreateQRToken] Failed to insert: %v", err)
		return "", 0, err
	}

	log.Printf("✅ [CreateQRToken] Created: programId=%s, type=%s, expires=%ds", programId, qrType, QR_TOKEN_EXPIRY)
	return token, expiresAt, nil
}

// ============================================
// Claim Token Management (อายุ 10 นาที)
// ============================================

// ClaimQRTokenAnonymous สร้าง Claim Token โดยไม่ต้อง Login
// ใช้เมื่อ Student scan QR ครั้งแรก (ก่อน Login)
func ClaimQRTokenAnonymous(token string) (string, *models.QRToken, error) {
	ctx := context.TODO()
	now := time.Now()

	log.Printf("🔍 [ClaimAnonymous] Token: %s", token)

	// ตรวจสอบ QR Token
	qrToken, err := findValidQRToken(ctx, token, now)
	if err != nil {
		return "", nil, err
	}

	// สร้าง Claim Token (ยังไม่มี StudentID)
	claimToken, err := createClaimToken(ctx, token, qrToken.ProgramID, qrToken.Type, nil)
	if err != nil {
		return "", nil, err
	}

	log.Printf("✅ [ClaimAnonymous] Success: claimToken=%s", claimToken)
	return claimToken, qrToken, nil
}

// ClaimQRToken สร้าง Claim Token สำหรับ Student ที่ Login แล้ว (Legacy)
func ClaimQRToken(token, studentId string) (*models.QRToken, error) {
	ctx := context.TODO()
	now := time.Now()

	log.Printf("🔍 [ClaimQRToken] Token: %s, StudentId: %s", token, studentId)

	// ตรวจสอบ QR Token
	qrToken, err := findValidQRToken(ctx, token, now)
	if err != nil {
		return nil, err
	}

	// ตรวจสอบ Enrollment (ถ้ามี studentId)
	var studentObjID *primitive.ObjectID
	if studentId != "" {
		objID, err := convertToObjectID(studentId)
		if err != nil {
			return nil, err
		}
		studentObjID = &objID

		// ตรวจสอบว่าลงทะเบียนหรือไม่
		if err := checkStudentEnrollment(studentId, qrToken.ProgramID.Hex()); err != nil {
			return nil, err
		}
	}

	// สร้าง Claim Token
	_, err = createClaimToken(ctx, token, qrToken.ProgramID, qrToken.Type, studentObjID)
	if err != nil {
		return nil, err
	}

	qrToken.ClaimedByStudentID = studentObjID
	log.Printf("✅ [ClaimQRToken] Success")
	return qrToken, nil
}

// ValidateClaimToken ตรวจสอบ Claim Token (หลัง Login)
func ValidateClaimToken(claimToken, studentId string) (*models.QRTokenClaim, error) {
	ctx := context.TODO()
	now := time.Now()

	log.Printf("🔍 [ValidateClaimToken] ClaimToken: %s, StudentId: %s", claimToken, studentId)

	// หา Claim Token
	claim, err := findValidClaimToken(ctx, claimToken, now)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ [ValidateClaimToken] Found: programId=%s, type=%s", claim.ProgramID.Hex(), claim.Type)

	// ถ้ายังไม่มี StudentID → อัปเดต (กรณี Scan ก่อน Login)
	if claim.StudentID == nil && studentId != "" {
		if err := updateClaimTokenWithStudent(ctx, claimToken, studentId, claim.ProgramID.Hex()); err != nil {
			return nil, err
		}

		// อัปเดต claim object
		objID, _ := convertToObjectID(studentId)
		claim.StudentID = &objID
	}

	// ถ้ามี StudentID แล้ว → ตรวจสอบว่าตรงกันหรือไม่
	if claim.StudentID != nil && studentId != "" {
		studentObjID, _ := convertToObjectID(studentId)
		if claim.StudentID.Hex() != studentObjID.Hex() {
			log.Printf("❌ [ValidateClaimToken] Token belongs to different student")
			return nil, fmt.Errorf("claim Token นี้ไม่ได้เป็นของคุณ")
		}
	}

	log.Printf("✅ [ValidateClaimToken] Validation successful")
	return claim, nil
}

// MarkClaimTokenAsUsed ทำเครื่องหมาย Claim Token ว่าใช้แล้ว
func MarkClaimTokenAsUsed(claimToken string) error {
	ctx := context.TODO()
	log.Printf("🔒 [MarkAsUsed] ClaimToken: %s", claimToken)

	_, err := DB.QrClaimCollection.UpdateOne(ctx, bson.M{
		"claimToken": claimToken,
	}, bson.M{
		"$set": bson.M{"used": true},
	})

	if err != nil {
		log.Printf("❌ [MarkAsUsed] Failed: %v", err)
		return err
	}

	log.Printf("✅ [MarkAsUsed] Success")
	return nil
}

// ValidateQRToken ตรวจสอบ QR Token (Legacy - สำหรับระบบเก่า)
func ValidateQRToken(token, studentId string) (*models.QRToken, error) {
	ctx := context.TODO()
	studentObjID, err := convertToObjectID(studentId)
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

	err = DB.QrClaimCollection.FindOne(ctx, bson.M{
		"token":     token,
		"studentId": studentObjID,
		"expireAt":  bson.M{"$gt": time.Now()},
	}).Decode(&claim)

	if err != nil {
		return nil, fmt.Errorf("QR token not claimed or expired: %v", err)
	}

	// ตรวจสอบ Enrollment
	if err := checkStudentEnrollment(studentId, claim.ProgramID.Hex()); err != nil {
		return nil, err
	}

	return &models.QRToken{
		Token:              claim.Token,
		ProgramID:          claim.ProgramID,
		Type:               claim.Type,
		ClaimedByStudentID: &studentObjID,
	}, nil
}

// ============================================
// Private Helper Functions
// ============================================

// findValidQRToken หา QR Token ที่ยังไม่หมดอายุ
func findValidQRToken(ctx context.Context, token string, now time.Time) (*models.QRToken, error) {
	var qrToken models.QRToken
	err := DB.QrTokenCollection.FindOne(ctx, bson.M{
		"token":     token,
		"expiresAt": bson.M{"$gt": now.Unix()},
	}).Decode(&qrToken)

	if err != nil {
		log.Printf("❌ QR Token expired or invalid: %s", token)
		return nil, fmt.Errorf("QR Code หมดอายุ กรุณาสแกนใหม่")
	}

	log.Printf("✅ QR Token found: programId=%s, type=%s", qrToken.ProgramID.Hex(), qrToken.Type)
	return &qrToken, nil
}

// findValidClaimToken หา Claim Token ที่ยังไม่หมดอายุและยังไม่ใช้
func findValidClaimToken(ctx context.Context, claimToken string, now time.Time) (*models.QRTokenClaim, error) {
	var claim models.QRTokenClaim
	err := DB.QrClaimCollection.FindOne(ctx, bson.M{
		"claimToken": claimToken,
		"expiresAt":  bson.M{"$gt": now},
		"used":       false,
	}).Decode(&claim)

	if err != nil {
		log.Printf("❌ Claim Token expired or not found: %s", claimToken)
		return nil, fmt.Errorf("session หมดอายุ กรุณาสแกน QR ใหม่")
	}

	return &claim, nil
}

// createClaimToken สร้าง Claim Token ใหม่
func createClaimToken(ctx context.Context, originalToken string, programID primitive.ObjectID, qrType string, studentID *primitive.ObjectID) (string, error) {
	claimToken := uuid.NewString()
	now := time.Now()
	expiresAt := now.Add(time.Duration(CLAIM_TOKEN_EXPIRY) * time.Second)

	claim := models.QRTokenClaim{
		ClaimToken:    claimToken,
		OriginalToken: originalToken,
		ProgramID:     programID,
		Type:          qrType,
		StudentID:     studentID,
		CreatedAt:     now,
		ExpiresAt:     expiresAt,
		Used:          false,
	}

	_, err := DB.QrClaimCollection.InsertOne(ctx, claim)
	if err != nil {
		log.Printf("❌ Failed to create claim token: %v", err)
		return "", fmt.Errorf("ไม่สามารถสร้าง Claim Token ได้")
	}

	log.Printf("✅ Claim Token created: %s, expires at: %s", claimToken, expiresAt.Format("15:04:05"))
	return claimToken, nil
}

// updateClaimTokenWithStudent อัปเดต Claim Token ด้วย StudentID
func updateClaimTokenWithStudent(ctx context.Context, claimToken, studentId, programId string) error {
	log.Printf("🔄 Updating claim token with studentId: %s", studentId)

	studentObjID, err := convertToObjectID(studentId)
	if err != nil {
		return err
	}

	// ตรวจสอบ Enrollment
	if err := checkStudentEnrollment(studentId, programId); err != nil {
		return err
	}

	// อัปเดต StudentID
	_, err = DB.QrClaimCollection.UpdateOne(ctx, bson.M{
		"claimToken": claimToken,
	}, bson.M{
		"$set": bson.M{"studentId": studentObjID},
	})

	if err != nil {
		log.Printf("❌ Failed to update claim token: %v", err)
		return fmt.Errorf("ไม่สามารถอัปเดตข้อมูลได้")
	}

	log.Printf("✅ Claim token updated")
	return nil
}

// checkStudentEnrollment ตรวจสอบว่า Student ลงทะเบียนกิจกรรมหรือไม่
func checkStudentEnrollment(studentId, programId string) error {
	itemIDs, found := enrollments.FindEnrolledItems(studentId, programId)
	if !found || len(itemIDs) == 0 {
		log.Printf("❌ Student not enrolled: %s", studentId)
		return fmt.Errorf("คุณไม่ได้ลงทะเบียนกิจกรรมนี้")
	}
	log.Printf("✅ Student enrolled in %d items", len(itemIDs))
	return nil
}
