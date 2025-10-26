package hourhistory

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"log"
	"strings"
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
// ตรวจสอบและให้ชั่วโมงตอน program success (complete) แทน (ใน VerifyAndGrantHours)

// VerifyAndGrantHours ตรวจสอบและให้ชั่วโมงเมื่อกิจกรรมเสร็จสิ้น (trigger เมื่อ program success/complete)
// Logic ใหม่:
// - เช็คว่ามี check-in/out ครบทุกวันตาม programItem.Dates หรือไม่
// - เช็คว่าเวลา check-in อยู่ในช่วงที่กำหนด (±30 นาที) หรือไม่
// - เข้าร่วมครบทุกวัน + ตรงเวลาทุกวัน = attended + ได้ชั่วโมงเต็ม
// - เข้าร่วมไม่ครบ หรือมาสาย = attended + 0 ชั่วโมง
// - ไม่มาเลย = absent + 0 ชั่วโมง
func VerifyAndGrantHours(
	ctx context.Context,
	enrollmentID primitive.ObjectID,
) error {
	loc, _ := time.LoadLocation("Asia/Bangkok")

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
	if totalDays == 0 {
		return fmt.Errorf("program item has no dates")
	}

	// 3) หา HourChangeHistory record
	var hourRecord models.HourChangeHistory
	err = DB.HourChangeHistoryCollection.FindOne(ctx, bson.M{
		"enrollmentId": enrollmentID,
		"sourceType":   "program",
		"sourceId":     enrollment.ProgramID,
	}).Decode(&hourRecord)

	if err != nil {
		// ไม่เจอ record → skip
		log.Printf("⚠️ No hour record found for enrollment %s", enrollmentID.Hex())
		return nil
	}

	// 4) สร้าง map ของ checkin/checkout records ตามวันที่
	checkinoutMap := make(map[string]models.CheckinoutRecord)
	if enrollment.CheckinoutRecord != nil {
		for _, record := range *enrollment.CheckinoutRecord {
			var dateKey string
			if record.Checkin != nil {
				dateKey = record.Checkin.In(loc).Format("2006-01-02")
			} else if record.Checkout != nil {
				dateKey = record.Checkout.In(loc).Format("2006-01-02")
			}
			if dateKey != "" {
				checkinoutMap[dateKey] = record
			}
		}
	}

	// 5) วิเคราะห์แต่ละวันใน programItem.Dates
	daysOnTime := 0     // วันที่มา check-in/out ตรงเวลา
	daysLate := 0       // วันที่มา check-in/out แต่สาย
	daysIncomplete := 0 // วันที่มีแต่ checkin หรือ checkout อย่างเดียว
	daysAbsent := 0     // วันที่ไม่มา

	missingDates := []string{}    // วันที่ไม่มาเลย
	lateDates := []string{}       // วันที่มาแต่สาย
	incompleteDates := []string{} // วันที่เช็คไม่ครบ

	log.Printf("🔍 [DEBUG] Enrollment %s - Starting verification for %d days", enrollmentID.Hex(), totalDays)
	log.Printf("🔍 [DEBUG] Total checkinout records: %d", len(checkinoutMap))

	for idx, programDate := range programItem.Dates {
		dateKey := programDate.Date
		record, hasRecord := checkinoutMap[dateKey]

		log.Printf("🔍 [DEBUG] Day %d/%d - Date: %s", idx+1, totalDays, dateKey)
		log.Printf("🔍 [DEBUG]   ├─ Activity Time: %s - %s", programDate.Stime, programDate.Etime)

		if !hasRecord || (record.Checkin == nil && record.Checkout == nil) {
			// ไม่มา check-in/out เลย
			log.Printf("🔍 [DEBUG]   └─ ❌ ABSENT - No check-in/out record")
			daysAbsent++
			missingDates = append(missingDates, dateKey)
			continue
		}

		// มี record แล้ว - แสดงเวลาที่เช็ค
		checkinStr := "N/A"
		checkoutStr := "N/A"
		if record.Checkin != nil {
			checkinStr = record.Checkin.In(loc).Format("15:04:05")
		}
		if record.Checkout != nil {
			checkoutStr = record.Checkout.In(loc).Format("15:04:05")
		}
		log.Printf("🔍 [DEBUG]   ├─ Check-in: %s, Check-out: %s", checkinStr, checkoutStr)

		// เช็คว่ามีทั้ง checkin และ checkout หรือไม่
		if record.Checkin == nil || record.Checkout == nil {
			// มีแต่ checkin หรือ checkout อย่างเดียว
			log.Printf("🔍 [DEBUG]   └─ ⚠️ INCOMPLETE - Missing check-in or check-out")
			daysIncomplete++
			incompleteDates = append(incompleteDates, dateKey)
			continue
		}

		// มีทั้ง checkin และ checkout แล้ว → เช็คเวลา
		if programDate.Stime != "" {
			// Parse เวลาเริ่มกิจกรรม
			startTime, err := time.ParseInLocation("2006-01-02 15:04", programDate.Date+" "+programDate.Stime, loc)
			if err == nil {
				// อนุญาตเช็คอินก่อนเวลา 30 นาที และหลังเวลา 30 นาที
				earlyLimit := startTime.Add(-30 * time.Minute)
				lateLimit := startTime.Add(30 * time.Minute)
				checkinTime := record.Checkin.In(loc)

				log.Printf("🔍 [DEBUG]   ├─ Activity Start: %s", startTime.Format("15:04:05"))
				log.Printf("🔍 [DEBUG]   ├─ Allowed Range: %s - %s (±30 min)", earlyLimit.Format("15:04:05"), lateLimit.Format("15:04:05"))
				log.Printf("🔍 [DEBUG]   ├─ Actual Check-in: %s", checkinTime.Format("15:04:05"))

				if (checkinTime.Equal(earlyLimit) || checkinTime.After(earlyLimit)) &&
					(checkinTime.Before(lateLimit) || checkinTime.Equal(lateLimit)) {
					// เช็คอินตรงเวลา (±30 นาที)
					log.Printf("🔍 [DEBUG]   └─ ✅ ON TIME - Within allowed range")
					daysOnTime++
				} else {
					// เช็คอินไม่ตรงเวลา (เร็วเกิน หรือ สายเกิน)
					if checkinTime.Before(earlyLimit) {
						diff := earlyLimit.Sub(checkinTime)
						log.Printf("🔍 [DEBUG]   └─ ⚠️ TOO EARLY - %d minutes before allowed time", int(diff.Minutes()))
					} else {
						diff := checkinTime.Sub(lateLimit)
						log.Printf("🔍 [DEBUG]   └─ ⚠️ TOO LATE - %d minutes after allowed time", int(diff.Minutes()))
					}
					daysLate++
					lateDates = append(lateDates, dateKey)
				}
			} else {
				// ถ้า parse เวลาไม่ได้ ถือว่ามา (ให้ประโยชน์ของข้อสงสัย)
				log.Printf("🔍 [DEBUG]   └─ ✅ ON TIME - No time specified or parse error")
				daysOnTime++
			}
		} else {
			// ถ้าไม่มีเวลากำหนด ถือว่ามา
			log.Printf("🔍 [DEBUG]   └─ ✅ ON TIME - No specific time required")
			daysOnTime++
		}
	}

	totalValidDays := daysOnTime + daysLate + daysIncomplete
	hasAttendedAllDays := (daysOnTime == totalDays) // ต้องมาตรงเวลาครบทุกวัน

	log.Printf("🔍 [DEBUG] Summary:")
	log.Printf("🔍 [DEBUG]   ├─ Total Days Required: %d", totalDays)
	log.Printf("🔍 [DEBUG]   ├─ Days On Time: %d", daysOnTime)
	log.Printf("🔍 [DEBUG]   ├─ Days Late: %d", daysLate)
	log.Printf("🔍 [DEBUG]   ├─ Days Incomplete: %d", daysIncomplete)
	log.Printf("🔍 [DEBUG]   ├─ Days Absent: %d", daysAbsent)
	log.Printf("🔍 [DEBUG]   └─ Has Attended All Days: %v", hasAttendedAllDays)

	var newStatus string
	var newHourChange int
	var newRemark string

	// 6) Logic การให้ชั่วโมง
	if daysAbsent == totalDays {
		// ❌ ไม่มาเข้าร่วมเลยทุกวัน
		newStatus = models.HCStatusAbsent
		newHourChange = -*programItem.Hour
		newRemark = fmt.Sprintf("❌ ไม่มาเข้าร่วมกิจกรรมเลย (0/%d วัน)", totalDays)
	} else if hasAttendedAllDays {
		// ✅ มาครบทุกวัน และ ตรงเวลาทุกวัน → ได้ชั่วโมงเต็ม
		newStatus = models.HCStatusAttended
		newHourChange = *programItem.Hour
		newRemark = fmt.Sprintf("✅ เข้าร่วมครบถ้วนและตรงเวลาทุกวัน (%d/%d วัน) - ได้รับ %d ชั่วโมง", daysOnTime, totalDays, newHourChange)
	} else {
		// ⚠️ มาแต่ไม่ครบ หรือมาสาย หรือเช็คไม่ครบ → attended แต่ไม่ได้ชั่วโมง
		newStatus = models.HCStatusAttended
		newHourChange = 0

		// สร้าง remark ที่ละเอียด
		details := []string{}
		if daysOnTime > 0 {
			details = append(details, fmt.Sprintf("ตรงเวลา %d วัน", daysOnTime))
		}
		if daysLate > 0 {
			details = append(details, fmt.Sprintf("สาย %d วัน", daysLate))
		}
		if daysIncomplete > 0 {
			details = append(details, fmt.Sprintf("เช็คไม่ครบ %d วัน", daysIncomplete))
		}
		if daysAbsent > 0 {
			details = append(details, fmt.Sprintf("ขาด %d วัน", daysAbsent))
		}

		detailsStr := ""
		if len(details) > 0 {
			detailsStr = " (" + joinStrings(details, ", ") + ")"
		}

		newRemark = fmt.Sprintf("⚠️ เข้าร่วม %d/%d วัน%s - ไม่ได้รับชั่วโมง", totalValidDays, totalDays, detailsStr)

		// เพิ่มรายละเอียดวันที่มีปัญหา (ถ้ามี)
		if len(missingDates) > 0 && len(missingDates) <= 3 {
			newRemark += fmt.Sprintf(" | ขาดวันที่: %s", joinStrings(missingDates, ", "))
		}
		if len(lateDates) > 0 && len(lateDates) <= 3 {
			newRemark += fmt.Sprintf(" | สายวันที่: %s", joinStrings(lateDates, ", "))
		}
		if len(incompleteDates) > 0 && len(incompleteDates) <= 3 {
			newRemark += fmt.Sprintf(" | เช็คไม่ครบวันที่: %s", joinStrings(incompleteDates, ", "))
		}
	}

	// 7) อัปเดต HourChangeHistory
	filter := bson.M{
		"enrollmentId": enrollmentID,
		"sourceType":   "program",
		"sourceId":     enrollment.ProgramID,
	}

	update := bson.M{
		"$set": bson.M{
			"status":     newStatus,
			"hourChange": newHourChange,
			"remark":     newRemark,
			"changeAt":   time.Now(),
		},
	}

	log.Printf("� [DEBUG] Final Decision:")
	log.Printf("🔍 [DEBUG]   ├─ Status: %s", newStatus)
	log.Printf("🔍 [DEBUG]   ├─ Hours Granted: %d", newHourChange)
	log.Printf("🔍 [DEBUG]   └─ Remark: %s", newRemark)
	log.Printf("📝 Updating hour change history for enrollment %s: status=%s, hours=%d",
		enrollmentID.Hex(), newStatus, newHourChange)

	_, err = DB.HourChangeHistoryCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to verify and grant hours: %v", err)
	}

	// 🔄 Update student status หลังจากมีการเปลี่ยนแปลงชั่วโมง
	if err := updateStudentStatus(ctx, enrollment.StudentID); err != nil {
		log.Printf("⚠️ Warning: Failed to update student status for %s: %v", enrollment.StudentID.Hex(), err)
		// ไม่ return error เพราะการอัปเดตชั่วโมงสำเร็จแล้ว เหลือแค่ status
	}

	return nil
}

// joinStrings รวม string slice ด้วย separator
func joinStrings(arr []string, sep string) string {
	if len(arr) == 0 {
		return ""
	}
	result := arr[0]
	for i := 1; i < len(arr); i++ {
		result += sep + arr[i]
	}
	return result
}

// ProcessEnrollmentsForCompletedProgram processes all enrollments for a program
// that has been marked as complete. This is an exported helper so other
// packages (jobs, programs service, admin handlers) can call the same logic
// used by the background worker.
func ProcessEnrollmentsForCompletedProgram(ctx context.Context, programID primitive.ObjectID) error {
	log.Println("📝 Processing enrollments for completed program (hour-history): ++++++++++++++++", programID.Hex())

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
		if err := VerifyAndGrantHours(ctx, enrollment.ID); err != nil {
			log.Printf("⚠️ Failed to verify hours for enrollment %s: %v", enrollment.ID.Hex(), err)
			errorCount++
		} else {
			successCount++
		}
	}

	// log.Printf("✅ Processed %d enrollments successfully, %d errors", successCount, errorCount)
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

// GetStudentHoursSummary คำนวณชั่วโมงรวมของนิสิตจาก hour history
// รวมทั้ง attended (บวก) และ absent (ลบ) เพื่อคำนวณชั่วโมงที่แท้จริง
func GetStudentHoursSummary(ctx context.Context, studentID primitive.ObjectID) (map[string]interface{}, error) {
	// Aggregate pipeline เพื่อรวมชั่วโมงตาม skillType
	// รวมทั้ง attended และ absent (absent จะมี hourChange เป็นลบ)
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"studentId": studentID,
				"status": bson.M{
					"$in": []string{models.HCStatusAttended, models.HCStatusAbsent}, // รวมทั้ง attended และ absent
				},
			},
		},
		{
			"$group": bson.M{
				"_id": "$skillType", // group ตาม soft/hard
				"totalHours": bson.M{
					"$sum": "$hourChange", // รวมชั่วโมง (attended = +, absent = -)
				},
			},
		},
	}

	cursor, err := DB.HourChangeHistoryCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถคำนวณชั่วโมงรวมได้: %v", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("ไม่สามารถถอดรหัสผลลัพธ์ได้: %v", err)
	}

	// สร้าง summary object
	summary := map[string]interface{}{
		"softSkill": 0,
		"hardSkill": 0,
	}

	// Map ผลลัพธ์จาก aggregation
	for _, result := range results {
		skillType, _ := result["_id"].(string)
		totalHours, _ := result["totalHours"].(int32)

		if skillType == "soft" {
			summary["softSkill"] = int(totalHours)
		} else if skillType == "hard" {
			summary["hardSkill"] = int(totalHours)
		}
	}

	return summary, nil
}

// ========================================
// Student Status Management
// ========================================

// UpdateStudentStatus - คำนวณและอัปเดตสถานะของนักศึกษาตามชั่วโมงสุทธิที่ได้รับจาก HourChangeHistory
// Exported เพื่อให้ packages อื่น (certificates, students) เรียกใช้ได้
func UpdateStudentStatus(ctx context.Context, studentID primitive.ObjectID) error {
	// 1) ดึงข้อมูล student (ฐานชั่วโมง)
	var student models.Student
	if err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student); err != nil {
		return fmt.Errorf("student not found: %v", err)
	}

	// 2) คำนวณชั่วโมงสุทธิจาก HourChangeHistory
	softNet, hardNet, err := CalculateNetHours(ctx, studentID, student.SoftSkill, student.HardSkill)
	if err != nil {
		return err
	}

	// 3) คำนวณสถานะใหม่จาก "สุทธิ"
	newStatus := CalculateStatus(softNet, hardNet)

	// 4) อัปเดตสถานะ (ถ้าเปลี่ยนแปลง)
	if student.Status != newStatus {
		update := bson.M{"$set": bson.M{"status": newStatus}}
		if _, err := DB.StudentCollection.UpdateOne(ctx, bson.M{"_id": studentID}, update); err != nil {
			return fmt.Errorf("failed to update student status: %v", err)
		}

		log.Printf("✅ [UpdateStudentStatus] %s (%s) base(soft=%d,hard=%d) => net(soft=%d,hard=%d) => status: %d -> %d",
			student.ID.Hex(), student.Name, student.SoftSkill, student.HardSkill, softNet, hardNet, student.Status, newStatus)
	} else {
		log.Printf("ℹ️ [UpdateStudentStatus] %s (%s) status unchanged (status=%d, soft=%d, hard=%d)",
			student.ID.Hex(), student.Name, newStatus, softNet, hardNet)
	}

	return nil
}

// updateStudentStatus - internal wrapper (backward compatibility)
func updateStudentStatus(ctx context.Context, studentID primitive.ObjectID) error {
	return UpdateStudentStatus(ctx, studentID)
}

// CalculateNetHours - คำนวณชั่วโมงสุทธิจาก base hours + hour history delta
// Exported เพื่อให้ packages อื่นเรียกใช้ได้
func CalculateNetHours(ctx context.Context, studentID primitive.ObjectID, baseSoft, baseHard int) (softNet, hardNet int, err error) {
	pipeline := []bson.M{
		{"$match": bson.M{
			"studentId": studentID,
			"status": bson.M{"$in": []string{
				models.HCStatusAttended, models.HCStatusAbsent, models.HCStatusApproved,
			}},
		}},
		{"$addFields": bson.M{
			"deltaHours": bson.M{
				"$switch": bson.M{
					"branches": bson.A{
						bson.M{
							"case": bson.M{"$in": bson.A{"$status", bson.A{models.HCStatusAttended, models.HCStatusApproved}}},
							"then": bson.M{"$abs": bson.M{"$toInt": bson.M{"$ifNull": bson.A{"$hourChange", 0}}}},
						},
						bson.M{
							"case": bson.M{"$eq": bson.A{"$status", models.HCStatusAbsent}},
							"then": bson.M{
								"$multiply": bson.A{
									-1,
									bson.M{"$abs": bson.M{"$toInt": bson.M{"$ifNull": bson.A{"$hourChange", 0}}}},
								},
							},
						},
					},
					"default": 0,
				},
			},
		}},
		{"$group": bson.M{
			"_id":        "$skillType", // "soft" | "hard"
			"totalHours": bson.M{"$sum": "$deltaHours"},
		}},
	}

	cursor, aggErr := DB.HourChangeHistoryCollection.Aggregate(ctx, pipeline)
	if aggErr != nil {
		return 0, 0, fmt.Errorf("aggregate hour deltas error: %v", aggErr)
	}
	defer cursor.Close(ctx)

	type agg struct {
		ID         string `bson:"_id"`
		TotalHours int64  `bson:"totalHours"`
	}
	var aggRows []agg
	if aggErr := cursor.All(ctx, &aggRows); aggErr != nil {
		return 0, 0, fmt.Errorf("aggregate decode error: %v", aggErr)
	}

	// บวกผลรวมสุทธิกับฐานชั่วโมงใน student
	softNet = baseSoft
	hardNet = baseHard
	for _, r := range aggRows {
		switch strings.ToLower(r.ID) {
		case "soft":
			softNet += int(r.TotalHours)
		case "hard":
			hardNet += int(r.TotalHours)
		}
	}

	return softNet, hardNet, nil
}

// CalculateStatus - คำนวณสถานะของนักศึกษาจากชั่วโมง soft skill และ hard skill
// Exported เพื่อให้ packages อื่นเรียกใช้ได้
func CalculateStatus(softSkill, hardSkill int) int {
	total := softSkill + hardSkill

	switch {
	case softSkill >= 30 && hardSkill >= 12:
		return 3 // ครบ
	case total >= 20:
		return 2 // น้อย
	default:
		return 1 // น้อยมาก
	}
}
