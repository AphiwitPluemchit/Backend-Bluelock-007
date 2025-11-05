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

// CreateHourChangeHistory สร้างบันทึก HourChangeHistory (ฟังก์ชันหลักในการสร้าง record)
// Parameters:
//   - studentID: รหัสนักศึกษา
//   - sourceType: ประเภทแหล่งที่มา "program" | "certificate" | "manual"
//   - sourceID: รหัสของแหล่งที่มา (nil สำหรับ manual)
//   - skillType: ประเภททักษะ "soft" | "hard"
//   - status: สถานะ (upcoming, participating, attended, absent, approved, manual)
//   - hourChange: จำนวนชั่วโมงที่เปลี่ยนแปลง (บวก = เพิ่ม, ลบ = ลด)
//   - title: หัวข้อ
//   - remark: หมายเหตุ
//   - enrollmentID: รหัสการลงทะเบียน (optional, สำหรับ program)
//   - programItemID: รหัส program item (optional, สำหรับ program)
func CreateHourChangeHistory(
	ctx context.Context,
	studentID primitive.ObjectID,
	sourceType string,
	sourceID *primitive.ObjectID,
	skillType string,
	status string,
	hourChange int,
	title string,
	remark string,
	enrollmentID *primitive.ObjectID,
	programItemID *primitive.ObjectID,
) (*models.HourChangeHistory, error) {
	history := models.HourChangeHistory{
		ID:            primitive.NewObjectID(),
		StudentID:     studentID,
		SourceType:    sourceType,
		SourceID:      sourceID,
		SkillType:     skillType,
		Status:        status,
		HourChange:    hourChange,
		Title:         title,
		Remark:        remark,
		ChangeAt:      time.Now(),
		EnrollmentID:  enrollmentID,
		ProgramItemID: programItemID,
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

// RecordCheckinActivity บันทึกการเช็คอินเข้าร่วมกิจกรรม
// อัปเดต status: HCStatusUpcoming → HCStatusParticipating (กำลังเข้าร่วม)
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
			"hourChange": 0,
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
	hasAttendedAllDays := (totalValidDays == totalDays)   // มาครบทุกวัน (เช็คอิน+เช็คเอาท์ครบ)
	hasAttendedAllDaysOnTime := (daysOnTime == totalDays) // มาครบทุกวัน + ตรงเวลาทุกวัน

	log.Printf("🔍 [DEBUG] Summary:")
	log.Printf("🔍 [DEBUG]   ├─ Total Days Required: %d", totalDays)
	log.Printf("🔍 [DEBUG]   ├─ Days On Time: %d", daysOnTime)
	log.Printf("🔍 [DEBUG]   ├─ Days Late: %d", daysLate)
	log.Printf("🔍 [DEBUG]   ├─ Days Incomplete: %d", daysIncomplete)
	log.Printf("🔍 [DEBUG]   ├─ Days Absent: %d", daysAbsent)
	log.Printf("🔍 [DEBUG]   ├─ Has Attended All Days: %v", hasAttendedAllDays)
	log.Printf("🔍 [DEBUG]   └─ Has Attended All Days On Time: %v", hasAttendedAllDaysOnTime)

	var newStatus string
	var newHourChange int
	var newRemark string

	// 6) Logic การให้ชั่วโมง
	if daysAbsent == totalDays {
		// ❌ ไม่มาเข้าร่วมเลยทุกวัน
		newStatus = models.HCStatusAbsent
		newHourChange = -*programItem.Hour
		newRemark = fmt.Sprintf("❌ ไม่มาเข้าร่วมกิจกรรมเลย (0/%d วัน)", totalDays)
	} else if !hasAttendedAllDays {
		// ⚠️ มาไม่ครบทุกวัน (ขาดบางวัน หรือ เช็คไม่ครบบางวัน) → late และไม่ได้ชั่วโมง
		newStatus = models.HCStatusLate
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

		newRemark = fmt.Sprintf("⚠️ เข้าร่วมไม่ครบ %d/%d วัน%s - ไม่ได้รับชั่วโมง", totalValidDays, totalDays, detailsStr)

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
	} else if hasAttendedAllDaysOnTime {
		// ✅ มาครบทุกวัน + ตรงเวลาทุกวัน → ได้ชั่วโมงเต็ม
		newStatus = models.HCStatusAttended
		newHourChange = *programItem.Hour
		newRemark = fmt.Sprintf("✅ เข้าร่วมครบถ้วนและตรงเวลาทุกวัน (%d/%d วัน) - ได้รับ %d ชั่วโมง", daysOnTime, totalDays, newHourChange)
	} else {
		// ⚠️ มาครบทุกวันแล้ว แต่มีบางวันที่สาย/เช็คไม่ครบ → late และไม่ได้ชั่วโมง
		newStatus = models.HCStatusLate
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

		detailsStr := ""
		if len(details) > 0 {
			detailsStr = " (" + joinStrings(details, ", ") + ")"
		}

		newRemark = fmt.Sprintf("⚠️ เข้าร่วมครบทุกวัน แต่%s - ไม่ได้รับชั่วโมง", detailsStr)

		// เพิ่มรายละเอียดวันที่มีปัญหา (ถ้ามี)
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
	if err := UpdateStudentStatus(ctx, enrollment.StudentID); err != nil {
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

// PopulateHistoryDetails ดึงข้อมูล Program, ProgramItem, และ Certificate มา populate ใน histories
func PopulateHistoryDetails(ctx context.Context, histories []models.HourChangeHistory) error {
	for i := range histories {
		history := &histories[i]

		// Populate ตาม sourceType
		if history.SourceType == "program" {
			// ดึง Program
			var program models.Program
			err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": history.SourceID}).Decode(&program)
			if err == nil {
				history.Program = &program
			}

			// ดึง ProgramItem (ถ้ามี programItemId)
			if history.ProgramItemID != nil {
				var programItem models.ProgramItem
				err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": *history.ProgramItemID}).Decode(&programItem)
				if err == nil {
					history.ProgramItem = &programItem
				}
			}
		} else if history.SourceType == "certificate" {
			// ดึง Certificate
			var certificate models.UploadCertificate
			err := DB.UploadCertificateCollection.FindOne(ctx, bson.M{"_id": history.SourceID}).Decode(&certificate)
			if err == nil {
				history.Certificate = &certificate
			}
		}
	}

	return nil
}

// GetHistoryWithDetailsAndFilters ดึงประวัติการเปลี่ยนแปลงชั่วโมงพร้อม populate objects
func GetHistoryWithDetailsAndFilters(
	ctx context.Context,
	studentID *primitive.ObjectID,
	sourceType string,
	statuses []string,
	searchTitle string,
	limit int,
	skip int,
) ([]models.HourChangeHistory, int64, error) {
	// ใช้ GetHistoryWithFilters เดิม
	histories, totalCount, err := GetHistoryWithFilters(ctx, studentID, sourceType, statuses, searchTitle, limit, skip)
	if err != nil {
		return nil, 0, err
	}

	// Populate details
	if err := PopulateHistoryDetails(ctx, histories); err != nil {
		log.Printf("Warning: failed to populate history details: %v", err)
		// ไม่ return error เพราะข้อมูลหลักยังใช้ได้
	}

	return histories, totalCount, nil
}

// GetStudentHoursSummary คำนวณชั่วโมงรวมของนิสิตจาก hour history
// รวมทั้ง attended (บวก), approved, manual และ absent (ลบ) เพื่อคำนวณชั่วโมงที่แท้จริง
func GetStudentHoursSummary(ctx context.Context, studentID primitive.ObjectID) (map[string]interface{}, error) {
	// Aggregate pipeline เพื่อรวมชั่วโมงตาม skillType
	// รวมทั้ง attended, approved, manual และ absent (absent จะมี hourChange เป็นลบ)
	// late ไม่นับเพราะไม่ได้ชั่วโมง (hourChange = 0)
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"studentId": studentID,
				"status": bson.M{
					"$in": []string{models.HCStatusAttended, models.HCStatusAbsent, models.HCStatusApproved, models.HCStatusManual}, // รวมทั้ง attended, approved, manual และ absent (late ไม่รวมเพราะ hourChange = 0)
				},
			},
		},
		{
			"$group": bson.M{
				"_id": "$skillType", // group ตาม soft/hard
				"totalHours": bson.M{
					"$sum": "$hourChange", // รวมชั่วโมง (attended/approved/manual = +, absent = -)
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

// UpdateStudentStatus คำนวณและอัปเดตสถานะของนักศึกษาตามชั่วโมงจาก HourChangeHistory
// เรียกใช้โดย packages อื่น (certificates, students, programs)
func UpdateStudentStatus(ctx context.Context, studentID primitive.ObjectID) error {
	// ดึงข้อมูล student
	var student models.Student
	if err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student); err != nil {
		return fmt.Errorf("student not found: %v", err)
	}

	// คำนวณชั่วโมงจาก HourChangeHistory
	softNet, hardNet, err := CalculateNetHours(ctx, studentID)
	if err != nil {
		return err
	}

	// คำนวณสถานะใหม่
	newStatus := CalculateStatus(softNet, hardNet)

	// อัปเดตสถานะ (ถ้าเปลี่ยนแปลง)
	if student.Status != newStatus {
		update := bson.M{"$set": bson.M{"status": newStatus}}
		if _, err := DB.StudentCollection.UpdateOne(ctx, bson.M{"_id": studentID}, update); err != nil {
			return fmt.Errorf("failed to update student status: %v", err)
		}

		log.Printf("✅ [UpdateStudentStatus] %s (%s) hours(soft=%d,hard=%d) => status: %d -> %d",
			student.ID.Hex(), student.Name, softNet, hardNet, student.Status, newStatus)
	} else {
		log.Printf("ℹ️ [UpdateStudentStatus] %s (%s) status unchanged (status=%d, soft=%d, hard=%d)",
			student.ID.Hex(), student.Name, newStatus, softNet, hardNet)
	}

	return nil
}

// CalculateNetHours คำนวณชั่วโมงรวมจาก HourChangeHistory
// รวมเฉพาะ status: attended (+), approved (+), manual (+), absent (-)
// late ไม่รวมเพราะ hourChange = 0
func CalculateNetHours(ctx context.Context, studentID primitive.ObjectID) (softNet, hardNet int, err error) {
	pipeline := []bson.M{
		{"$match": bson.M{
			"studentId": studentID,
			"status": bson.M{"$in": []string{
				models.HCStatusAttended, models.HCStatusAbsent, models.HCStatusApproved, models.HCStatusManual,
			}},
		}},
		{"$addFields": bson.M{
			"deltaHours": bson.M{
				"$switch": bson.M{
					"branches": bson.A{
						bson.M{
							"case": bson.M{"$in": bson.A{"$status", bson.A{models.HCStatusAttended, models.HCStatusApproved, models.HCStatusManual}}},
							"then": bson.M{"$toInt": bson.M{"$ifNull": bson.A{"$hourChange", 0}}},
						},
						bson.M{
							"case": bson.M{"$eq": bson.A{"$status", models.HCStatusAbsent}},
							"then": bson.M{
								"$multiply": bson.A{
									-1,
									bson.M{"$toInt": bson.M{"$ifNull": bson.A{"$hourChange", 0}}},
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

	// รวมชั่วโมงตาม skillType
	softNet = 0
	hardNet = 0
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

// CalculateStatus คำนวณสถานะนักศึกษาจากชั่วโมง soft และ hard skill
// Return: 1 = น้อยมาก, 2 = น้อย, 3 = ครบ
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
