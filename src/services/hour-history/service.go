package hourhistory

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ========================================
// Core Function - สร้าง HourChangeHistory
// ========================================

// SaveHourHistory บันทึกประวัติการเปลี่ยนแปลงชั่วโมง
func SaveHourHistory(
	ctx context.Context,
	studentID primitive.ObjectID,
	skillType string, // "soft" | "hard"
	hourChange int, // บวก = เพิ่ม, ลบ = ลด
	title string,
	remark string,
	sourceType string, // "program" | "certificate"
	sourceID primitive.ObjectID,
	enrollmentID *primitive.ObjectID, // optional, สำหรับ program เท่านั้น
) error {
	history := models.HourChangeHistory{
		ID:           primitive.NewObjectID(),
		SkillType:    skillType,
		HourChange:   hourChange,
		Remark:       remark,
		ChangeAt:     time.Now(),
		Title:        title,
		StudentID:    studentID,
		EnrollmentID: enrollmentID,
		SourceType:   sourceType,
		SourceID:     sourceID,
	}

	if _, err := DB.HourChangeHistoryCollection.InsertOne(ctx, history); err != nil {
		return fmt.Errorf("failed to save hour change history: %v", err)
	}

	return nil
}

// CreateHourChangeHistory สร้างบันทึก HourChangeHistory พร้อม status
func CreateHourChangeHistory(
	ctx context.Context,
	studentID primitive.ObjectID,
	enrollmentID *primitive.ObjectID,
	sourceType string,
	sourceID primitive.ObjectID,
	skillType string,
	status string,
	hourChange int,
	title string,
	remark string,
) (*models.HourChangeHistory, error) {
	history := models.HourChangeHistory{
		ID:           primitive.NewObjectID(),
		SourceType:   sourceType,
		SourceID:     sourceID,
		SkillType:    skillType,
		Status:       status,
		HourChange:   hourChange,
		Remark:       remark,
		ChangeAt:     time.Now(),
		Title:        title,
		StudentID:    studentID,
		EnrollmentID: enrollmentID,
	}

	_, err := DB.HourChangeHistoryCollection.InsertOne(ctx, history)
	if err != nil {
		return nil, fmt.Errorf("failed to create hour change history: %v", err)
	}

	return &history, nil
}

// ========================================
// Program-specific Functions
// ========================================

// RecordEnrollmentHourChange บันทึกการเปลี่ยนแปลงชั่วโมงตอน Enroll (สร้างใหม่)
// status: HCStatusPending (รอเข้าร่วม)
func RecordEnrollmentHourChange(
	ctx context.Context,
	studentID primitive.ObjectID,
	enrollmentID primitive.ObjectID,
	programID primitive.ObjectID,
	programName string,
	skillType string,
	expectedHours int,
) error {
	// สร้าง record ใหม่ตอน enroll
	_, err := CreateHourChangeHistory(
		ctx,
		studentID,
		&enrollmentID,
		"program",
		programID,
		skillType,
		models.HCStatusPending, // รอเข้าร่วมกิจกรรม
		expectedHours,
		programName,
		"ลงทะเบียนกิจกรรม (รอเข้าร่วม)",
	)
	return err
}

// UpdateCheckinHourChange อัปเดต HourChangeHistory ตอน Checkin (UPDATE record เดิม)
// เปลี่ยน status: HCStatusPending → HCStatusAttended (เริ่มเข้าร่วม)
func UpdateCheckinHourChange(
	ctx context.Context,
	enrollmentID primitive.ObjectID,
	checkinDate string,
) error {
	// หา record ที่มี enrollmentId และ status = pending
	filter := bson.M{
		"enrollmentId": enrollmentID,
		"status":       models.HCStatusPending,
		"sourceType":   "program",
	}

	// อัปเดต status และ remark
	update := bson.M{
		"$set": bson.M{
			"status":   models.HCStatusAttended,
			"remark":   fmt.Sprintf("เช็คอินเข้าร่วมกิจกรรม (วันที่ %s)", checkinDate),
			"changeAt": time.Now(),
		},
	}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update checkin hour change: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no pending hour change record found for enrollmentId: %s", enrollmentID.Hex())
	}

	return nil
}

// UpdateCheckoutHourChange อัปเดต HourChangeHistory ตอน Checkout (UPDATE record เดิม)
// เปลี่ยน status ตามเงื่อนไข:
// - HCStatusAttended (checkin แล้ว + เข้าร่วมครบ) → ได้ชั่วโมง
// - HCStatusPartial (checkin แล้ว + ไม่ครบ หรือ ไม่ checkin แต่ checkout) → ไม่ได้ชั่วโมง
func UpdateCheckoutHourChange(
	ctx context.Context,
	enrollmentID primitive.ObjectID,
	attendedAllDays bool,
	totalHours int,
	checkoutDate string,
) error {
	// หา record เดิมเพื่อเช็ค status ปัจจุบัน
	var currentRecord models.HourChangeHistory
	err := DB.HourChangeHistoryCollection.FindOne(ctx, bson.M{
		"enrollmentId": enrollmentID,
		"sourceType":   "program",
		"status":       bson.M{"$in": []string{models.HCStatusPending, models.HCStatusAttended}},
	}).Decode(&currentRecord)

	if err != nil {
		return fmt.Errorf("no hour change record found for enrollmentId: %s", enrollmentID.Hex())
	}

	// ตรวจสอบว่า checkin แล้วหรือยัง
	hasCheckedIn := currentRecord.Status == models.HCStatusAttended

	var status string
	var hourChange int
	var remark string

	if hasCheckedIn && attendedAllDays {
		// ✅ กรณี: checkin แล้ว + เข้าร่วมครบ → ได้ชั่วโมง
		status = models.HCStatusAttended
		hourChange = totalHours
		remark = fmt.Sprintf("เช็คเอาท์สำเร็จ - เข้าร่วมครบถ้วน ได้รับ %d ชั่วโมง (วันที่ %s)", totalHours, checkoutDate)
	} else if hasCheckedIn && !attendedAllDays {
		// ⚠️ กรณี: checkin แล้ว + เข้าร่วมไม่ครบ → ไม่ได้ชั่วโมง
		status = models.HCStatusPartial
		hourChange = 0
		remark = fmt.Sprintf("เช็คเอาท์ - เข้าร่วมไม่ครบถ้วน ไม่ได้รับชั่วโมง (วันที่ %s)", checkoutDate)
	} else {
		// ⚠️ กรณี: ไม่ได้ checkin แต่ checkout → ไม่ได้ชั่วโมง
		status = models.HCStatusPartial
		hourChange = 0
		remark = fmt.Sprintf("เช็คเอาท์โดยไม่ได้เช็คอิน - ไม่ได้รับชั่วโมง (วันที่ %s)", checkoutDate)
	}

	// อัปเดต record
	filter := bson.M{
		"enrollmentId": enrollmentID,
		"sourceType":   "program",
		"status":       bson.M{"$in": []string{models.HCStatusPending, models.HCStatusAttended}},
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"hourChange": hourChange,
			"remark":     remark,
			"changeAt":   time.Now(),
		},
	}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update checkout hour change: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no hour change record found for enrollmentId: %s", enrollmentID.Hex())
	}

	return nil
}

// MarkAsAbsent ทำเครื่องหมายเป็นขาดเรียน (ถ้ากิจกรรมจบแล้วแต่ยังเป็น pending)
// เปลี่ยน status: HCStatusPending → HCStatusAbsent
func MarkAsAbsent(
	ctx context.Context,
	enrollmentID primitive.ObjectID,
) error {
	filter := bson.M{
		"enrollmentId": enrollmentID,
		"status":       models.HCStatusPending,
		"sourceType":   "program",
	}

	update := bson.M{
		"$set": bson.M{
			"status":     models.HCStatusAbsent,
			"hourChange": 0,
			"remark":     "ไม่เข้าร่วมกิจกรรม (ขาด)",
			"changeAt":   time.Now(),
		},
	}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark as absent: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no pending record found for enrollmentId: %s", enrollmentID.Hex())
	}

	return nil
}

// ========================================
// Query Functions
// ========================================

// GetHistoryByStudent ดึงประวัติการเปลี่ยนแปลงชั่วโมงของนิสิต
func GetHistoryByStudent(ctx context.Context, studentID primitive.ObjectID) ([]models.HourChangeHistory, error) {
	cursor, err := DB.HourChangeHistoryCollection.Find(ctx, bson.M{"studentId": studentID})
	if err != nil {
		return nil, fmt.Errorf("failed to get hour history: %v", err)
	}
	defer cursor.Close(ctx)

	var histories []models.HourChangeHistory
	if err := cursor.All(ctx, &histories); err != nil {
		return nil, fmt.Errorf("failed to decode hour history: %v", err)
	}

	return histories, nil
}

// GetHistoryByStudentWithLimit ดึงประวัติการเปลี่ยนแปลงชั่วโมงของนักเรียน พร้อม limit
func GetHistoryByStudentWithLimit(ctx context.Context, studentID primitive.ObjectID, limit int) ([]models.HourChangeHistory, error) {
	filter := bson.M{"studentId": studentID}
	opts := options.Find().SetSort(bson.D{{Key: "changeAt", Value: -1}})

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := DB.HourChangeHistoryCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงประวัติการเปลี่ยนแปลงชั่วโมงได้: %v", err)
	}
	defer cursor.Close(ctx)

	var histories []models.HourChangeHistory
	if err := cursor.All(ctx, &histories); err != nil {
		return nil, fmt.Errorf("ไม่สามารถถอดรหัสประวัติการเปลี่ยนแปลงชั่วโมงได้: %v", err)
	}

	return histories, nil
}

// GetHistoryBySource ดึงประวัติการเปลี่ยนแปลงชั่วโมงตาม source (program/certificate)
func GetHistoryBySource(ctx context.Context, sourceType string, sourceID primitive.ObjectID) ([]models.HourChangeHistory, error) {
	cursor, err := DB.HourChangeHistoryCollection.Find(ctx, bson.M{
		"sourceType": sourceType,
		"sourceId":   sourceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get hour history: %v", err)
	}
	defer cursor.Close(ctx)

	var histories []models.HourChangeHistory
	if err := cursor.All(ctx, &histories); err != nil {
		return nil, fmt.Errorf("failed to decode hour history: %v", err)
	}

	return histories, nil
}

// GetHistoryByProgram ดึงประวัติการเปลี่ยนแปลงชั่วโมงของกิจกรรม พร้อม limit
func GetHistoryByProgram(ctx context.Context, programID primitive.ObjectID, limit int) ([]models.HourChangeHistory, error) {
	filter := bson.M{"sourceType": "program", "sourceId": programID}
	opts := options.Find().SetSort(bson.D{{Key: "changeAt", Value: -1}})

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := DB.HourChangeHistoryCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงประวัติการเปลี่ยนแปลงชั่วโมงได้: %v", err)
	}
	defer cursor.Close(ctx)

	var histories []models.HourChangeHistory
	if err := cursor.All(ctx, &histories); err != nil {
		return nil, fmt.Errorf("ไม่สามารถถอดรหัสประวัติการเปลี่ยนแปลงชั่วโมงได้: %v", err)
	}

	return histories, nil
}

// GetHistorySummary สรุปประวัติการเปลี่ยนแปลงชั่วโมง
func GetHistorySummary(ctx context.Context, studentID primitive.ObjectID) (map[string]interface{}, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"studentId": studentID}},
		{"$group": bson.M{
			"_id":        "$status",
			"count":      bson.M{"$sum": 1},
			"totalHours": bson.M{"$sum": "$hourChange"},
		}},
	}

	cursor, err := DB.HourChangeHistoryCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงสรุปประวัติการเปลี่ยนแปลงชั่วโมงได้: %v", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("ไม่สามารถถอดรหัสสรุปประวัติการเปลี่ยนแปลงชั่วโมงได้: %v", err)
	}

	summary := map[string]interface{}{
		"totalRecords":  0,
		"totalAttended": 0,
		"totalPending":  0,
		"totalPartial":  0,
		"totalAbsent":   0,
	}

	for _, result := range results {
		status, _ := result["_id"].(string)
		count, _ := result["count"].(int32)
		totalHours, _ := result["totalHours"].(int32)

		summary["totalRecords"] = summary["totalRecords"].(int) + int(count)

		switch status {
		case models.HCStatusAttended:
			summary["totalAttended"] = int(totalHours)
		case models.HCStatusPending:
			summary["totalPending"] = int(count)
		case models.HCStatusPartial:
			summary["totalPartial"] = int(count)
		case models.HCStatusAbsent:
			summary["totalAbsent"] = int(count)
		}
	}

	return summary, nil
}
