package hourhistory

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"log"
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
// status: HCStatusUpcoming (กำลังมาถึง - รอเข้าร่วมกิจกรรม)
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
		models.HCStatusUpcoming, // กำลังมาถึง - รอเข้าร่วมกิจกรรม
		expectedHours,
		programName,
		"ลงทะเบียนกิจกรรม (กำลังมาถึง)",
	)
	return err
}

// UpdateCheckinHourChange - DEPRECATED: ใช้ RecordCheckinActivity แทน
// เก็บไว้เพื่อ backward compatibility
func UpdateCheckinHourChange(
	ctx context.Context,
	enrollmentID primitive.ObjectID,
	checkinDate string,
) error {
	return RecordCheckinActivity(ctx, enrollmentID, checkinDate)
}

// RecordCheckinActivity บันทึกการเช็คอินเข้าร่วมกิจกรรม (แต่ละวัน)
// เปลี่ยน status: HCStatusUpcoming → HCStatusParticipating (กำลังเข้าร่วม)
func RecordCheckinActivity(
	ctx context.Context,
	enrollmentID primitive.ObjectID,
	checkinDate string,
) error {
	filter := bson.M{
		"enrollmentId": enrollmentID,
		"status":       models.HCStatusUpcoming,
		"sourceType":   "program",
	}

	update := bson.M{
		"$set": bson.M{
			"status":     models.HCStatusParticipating,
			"hourChange": 0, // ยังไม่ได้ชั่วโมง
			"remark":     fmt.Sprintf("กำลังเข้าร่วมกิจกรรม - เช็คอินวันที่ %s", checkinDate),
			"changeAt":   time.Now(),
		},
	}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to record checkin activity: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no upcoming hour change record found for enrollmentId: %s", enrollmentID.Hex())
	}

	return nil
}

// UpdateCheckinToVerifying เก็บไว้เพื่อ backward compatibility
// ⚠️ DEPRECATED: ใช้ RecordCheckinActivity แทน
func UpdateCheckinToVerifying(
	ctx context.Context,
	enrollmentID primitive.ObjectID,
	checkinDate string,
) error {
	return RecordCheckinActivity(ctx, enrollmentID, checkinDate)
}

// ⚠️ DEPRECATED: Functions ด้านล่างนี้ไม่ใช้แล้ว เนื่องจาก logic ใหม่
// ตรวจสอบและให้ชั่วโมงตอน program complete แทน (ใน VerifyAndGrantHours)

// VerifyAndGrantHours ตรวจสอบและให้ชั่วโมงเมื่อกิจกรรมเสร็จสิ้น (trigger เมื่อ program complete)
// Logic ใหม่:
// - เข้าร่วมครบทุกวัน + ทำฟอร์ม = attended + ได้ชั่วโมง
// - เข้าร่วมไม่ครบ หรือไม่ทำฟอร์ม = attended + 0 ชั่วโมง (ยังเก็บ record ไว้)
// - ไม่มาเลย (ยัง upcoming/participating) = absent + ลบชั่วโมงที่เคยให้ไว้
func VerifyAndGrantHours(
	ctx context.Context,
	enrollmentID primitive.ObjectID,
	programID primitive.ObjectID,
	totalHours int,
) error {
	// 1) ดึง Enrollment
	var enrollment models.Enrollment
	err := DB.EnrollmentCollection.FindOne(ctx, bson.M{"_id": enrollmentID}).Decode(&enrollment)
	if err != nil {
		return fmt.Errorf("enrollment not found: %v", err)
	}

	// 2) ดึง ProgramItem เพื่อเช็คจำนวนวันทั้งหมด
	var programItem models.ProgramItem
	err = DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": enrollment.ProgramItemID}).Decode(&programItem)
	if err != nil {
		return fmt.Errorf("program item not found: %v", err)
	}

	totalDays := len(programItem.Dates)

	// 3) หา HourChangeHistory record
	var hourRecord models.HourChangeHistory
	err = DB.HourChangeHistoryCollection.FindOne(ctx, bson.M{
		"enrollmentId": enrollmentID,
		"sourceType":   "program",
		"sourceId":     programID,
	}).Decode(&hourRecord)

	if err != nil {
		// ไม่เจอ record → skip
		log.Printf("⚠️ No hour record found for enrollment %s", enrollmentID.Hex())
		return nil
	}

	// 4) เช็คสถานะปัจจุบัน
	currentStatus := hourRecord.Status

	// 5) ตรวจสอบว่าเช็คชื่อครบหรือไม่
	checkinoutRecords := []models.CheckinoutRecord{}
	if enrollment.CheckinoutRecord != nil {
		checkinoutRecords = *enrollment.CheckinoutRecord
	}

	validDays := 0
	for _, record := range checkinoutRecords {
		if record.Checkin != nil && record.Checkout != nil {
			validDays++
		}
	}

	hasAttendedAllDays := (validDays == totalDays)
	hasSubmittedForm := enrollment.SubmissionID != nil

	var newStatus string
	var newHourChange int
	var newRemark string

	// 6) Logic ตามที่ร้องขอ
	if currentStatus == models.HCStatusUpcoming || currentStatus == models.HCStatusParticipating {
		// ❌ ไม่มาเข้าร่วม หรือไม่ได้ check in เลย
		newStatus = models.HCStatusAbsent
		newHourChange = -hourRecord.HourChange // ลบชั่วโมงที่เคยให้ไว้ (ถ้ามี)
		newRemark = fmt.Sprintf("❌ ไม่มาเข้าร่วมกิจกรรม - ลบชั่วโมง %d ชั่วโมง", -newHourChange)
	} else {
		// มาเข้าร่วมแล้ว (participating status)
		if hasAttendedAllDays && hasSubmittedForm {
			// ✅ มาครบทุกวัน + ทำฟอร์มแล้ว → ได้ชั่วโมง
			newStatus = models.HCStatusAttended
			newHourChange = totalHours
			newRemark = fmt.Sprintf("✅ เข้าร่วมครบถ้วน (%d/%d วัน) และทำฟอร์มเสร็จสิ้น - ได้รับ %d ชั่วโมง", validDays, totalDays, totalHours)
		} else {
			// ⚠️ มาไม่ครบ หรือไม่ทำฟอร์ม → attended แต่ไม่ได้ชั่วโมง
			newStatus = models.HCStatusAttended
			newHourChange = 0
			if !hasAttendedAllDays && !hasSubmittedForm {
				newRemark = fmt.Sprintf("⚠️ เข้าร่วมไม่ครบ (%d/%d วัน) และไม่ได้ทำฟอร์ม - ไม่ได้รับชั่วโมง", validDays, totalDays)
			} else if !hasAttendedAllDays {
				newRemark = fmt.Sprintf("⚠️ เข้าร่วมไม่ครบ (%d/%d วัน) - ไม่ได้รับชั่วโมง", validDays, totalDays)
			} else {
				newRemark = fmt.Sprintf("⚠️ เข้าร่วมครบถ้วน (%d/%d วัน) แต่ไม่ได้ทำฟอร์ม - ไม่ได้รับชั่วโมง", validDays, totalDays)
			}
		}
	}

	// 7) อัปเดต HourChangeHistory
	filter := bson.M{
		"enrollmentId": enrollmentID,
		"sourceType":   "program",
		"sourceId":     programID,
	}

	update := bson.M{
		"$set": bson.M{
			"status":     newStatus,
			"hourChange": newHourChange,
			"remark":     newRemark,
			"changeAt":   time.Now(),
		},
	}

	_, err = DB.HourChangeHistoryCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to verify and grant hours: %v", err)
	}

	return nil
}

// ProcessEnrollmentsForCompletedProgram processes all enrollments for a program
// that has been marked as complete. This is an exported helper so other
// packages (jobs, programs service, admin handlers) can call the same logic
// used by the background worker.
func ProcessEnrollmentsForCompletedProgram(ctx context.Context, programID primitive.ObjectID) error {
	log.Println("📝 Processing enrollments for completed program (hour-history):", programID.Hex())

	// 1) หา Program เพื่อดึง totalHours
	var program struct {
		Hour *int `bson:"hour"`
	}
	err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programID}).Decode(&program)
	if err != nil {
		return err
	}

	totalHours := 0
	if program.Hour != nil {
		totalHours = *program.Hour
	}

	// 2) หา ProgramItems ทั้งหมดของ program นี้
	cursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var programItemIDs []primitive.ObjectID
	for cursor.Next(ctx) {
		var item struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&item); err != nil {
			continue
		}
		programItemIDs = append(programItemIDs, item.ID)
	}

	// 3) หา Enrollments ทั้งหมดที่เกี่ยวข้อง
	enrollCursor, err := DB.EnrollmentCollection.Find(ctx, bson.M{
		"programId":     programID,
		"programItemId": bson.M{"$in": programItemIDs},
	})
	if err != nil {
		return err
	}
	defer enrollCursor.Close(ctx)

	// 4) ประมวลผลแต่ละ enrollment
	successCount := 0
	errorCount := 0

	for enrollCursor.Next(ctx) {
		var enrollment struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := enrollCursor.Decode(&enrollment); err != nil {
			log.Printf("⚠️ Failed to decode enrollment: %v", err)
			errorCount++
			continue
		}

		// เรียกฟังก์ชันตรวจสอบและให้ชั่วโมง (ใช้ VerifyAndGrantHours ในแพ็กเกจนี้)
		if err := VerifyAndGrantHours(ctx, enrollment.ID, programID, totalHours); err != nil {
			log.Printf("⚠️ Failed to verify hours for enrollment %s: %v", enrollment.ID.Hex(), err)
			errorCount++
		} else {
			successCount++
		}
	}

	log.Printf("✅ Processed %d enrollments successfully, %d errors", successCount, errorCount)
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
		"totalRecords":       0,
		"totalAttended":      0,
		"totalUpcoming":      0,
		"totalParticipating": 0,
		"totalAbsent":        0,
	}

	for _, result := range results {
		status, _ := result["_id"].(string)
		count, _ := result["count"].(int32)
		totalHours, _ := result["totalHours"].(int32)

		summary["totalRecords"] = summary["totalRecords"].(int) + int(count)

		switch status {
		case models.HCStatusAttended:
			summary["totalAttended"] = int(totalHours)
		case models.HCStatusUpcoming:
			summary["totalUpcoming"] = int(count)
		case models.HCStatusParticipating:
			summary["totalParticipating"] = int(count)
		case models.HCStatusAbsent:
			summary["totalAbsent"] = int(count)
		}
	}

	return summary, nil
}

// GetHistoryWithFilters ดึงประวัติการเปลี่ยนแปลงชั่วโมงพร้อม filters
func GetHistoryWithFilters(
	ctx context.Context,
	studentID *primitive.ObjectID,
	sourceType string,
	statuses []string,
	searchTitle string,
	limit int,
	skip int,
) ([]models.HourChangeHistory, int64, error) {
	// สร้าง filter query
	filter := bson.M{}

	// Filter by studentID (optional)
	if studentID != nil {
		filter["studentId"] = *studentID
	}

	// Filter by sourceType (optional)
	if sourceType != "" {
		filter["sourceType"] = sourceType
	}

	// Filter by multiple statuses (optional)
	if len(statuses) > 0 {
		filter["status"] = bson.M{"$in": statuses}
	}

	// Search by title (optional, case-insensitive)
	if searchTitle != "" {
		filter["title"] = bson.M{"$regex": primitive.Regex{Pattern: searchTitle, Options: "i"}}
	}

	// Count total documents matching filter
	totalCount, err := DB.HourChangeHistoryCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("ไม่สามารถนับจำนวนประวัติได้: %v", err)
	}

	// Set options for pagination and sorting
	opts := options.Find().
		SetSort(bson.D{{Key: "changeAt", Value: -1}}).
		SetSkip(int64(skip))

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	// Execute query
	cursor, err := DB.HourChangeHistoryCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("ไม่สามารถดึงประวัติการเปลี่ยนแปลงชั่วโมงได้: %v", err)
	}
	defer cursor.Close(ctx)

	var histories []models.HourChangeHistory
	if err := cursor.All(ctx, &histories); err != nil {
		return nil, 0, fmt.Errorf("ไม่สามารถถอดรหัสประวัติการเปลี่ยนแปลงชั่วโมงได้: %v", err)
	}

	return histories, totalCount, nil
}
