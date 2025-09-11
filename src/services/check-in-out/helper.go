package checkInOut

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// processStudentHours processes hours for a single student
func processStudentHours(ctx context.Context, studentID primitive.ObjectID, activityItemID primitive.ObjectID, activityItem models.ActivityItem, skillType string) (*HourChangeResult, error) {
	// ดึงข้อมูลนักเรียนเพื่อหาชื่อและรหัสนักศึกษา
	var student models.Student
	err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student)
	if err != nil {
		return nil, fmt.Errorf("student not found: %v", err)
	}

	// ดึงข้อมูล check-in/out ของนักเรียนนี้
	cursor, err := DB.CheckinCollection.Find(ctx, bson.M{
		"studentId":      studentID,
		"activityItemId": activityItemID,
	})
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลบันทึกการเข้า/ออกได้: %v", err)
	}
	defer cursor.Close(ctx)

	var checkinRecords []models.CheckinRecord
	if err := cursor.All(ctx, &checkinRecords); err != nil {
		return nil, fmt.Errorf("ไม่สามารถถอดรหัสบันทึกการเช็คอิน/เช็คเอาท์ได้: %v", err)
	}

	// ถ้าไม่มี check-in/out records เลย ให้ลบ hour
	if len(checkinRecords) == 0 {
		err := removeStudentHours(ctx, studentID, *activityItem.Hour, skillType)
		if err != nil {
			return nil, err
		}

		// บันทึกประวัติการเปลี่ยนแปลง
		err = saveHourChangeHistory(ctx, studentID, student.Name, student.Code,
			activityItem.ActivityID.Hex(), "", activityItemID.Hex(), "", skillType,
			-*activityItem.Hour, "ไม่พบบันทึกการเช็คอิน/เช็คเอาท์")
		if err != nil {
			// ไม่ return error เพราะการบันทึกประวัติไม่ควรทำให้การประมวลผลล้มเหลว
			fmt.Printf("Warning: Failed to save hour change history: %v\n", err)
		}

		return &HourChangeResult{
			StudentID:   studentID.Hex(),
			StudentName: student.Name,
			StudentCode: student.Code,
			SkillType:   skillType,
			HoursChange: -*activityItem.Hour,
			Message:     "ไม่พบบันทึกการเช็คอิน/เช็คเอาท์ - ชั่วโมงถูกลบออก",
		}, nil
	}

	// จัดกลุ่ม check-in/out ตามวัน
	checkinByDate := make(map[string][]time.Time)
	checkoutByDate := make(map[string][]time.Time)

	for _, record := range checkinRecords {
		date := record.Timestamp.Format("2006-01-02")
		switch record.Type {
		case "checkin":
			checkinByDate[date] = append(checkinByDate[date], record.Timestamp)
		case "checkout":
			checkoutByDate[date] = append(checkoutByDate[date], record.Timestamp)
		}
	}

	// ประมวลผลแต่ละวัน
	totalHoursToAdd := 0
	processedDates := make(map[string]bool)

	for _, dateInfo := range activityItem.Dates {
		date := dateInfo.Date
		if processedDates[date] {
			continue
		}
		processedDates[date] = true

		// หา check-in และ checkout ที่เร็วที่สุดในวันนี้
		var earliestCheckin *time.Time
		var earliestCheckout *time.Time

		if checkins, exists := checkinByDate[date]; exists && len(checkins) > 0 {
			// หา check-in ที่เร็วที่สุด
			earliest := checkins[0]
			for _, ci := range checkins {
				if ci.Before(earliest) {
					earliest = ci
				}
			}
			earliestCheckin = &earliest
		}

		if checkouts, exists := checkoutByDate[date]; exists && len(checkouts) > 0 {
			// หา checkout ที่เร็วที่สุด
			earliest := checkouts[0]
			for _, co := range checkouts {
				if co.Before(earliest) {
					earliest = co
				}
			}
			earliestCheckout = &earliest
		}

		// ตรวจสอบเงื่อนไขการเพิ่มชั่วโมง
		hoursToAdd := calculateHoursForDate(dateInfo, earliestCheckin, earliestCheckout, *activityItem.Hour)
		totalHoursToAdd += hoursToAdd
	}

	// อัพเดทชั่วโมงของนักเรียน
	var message string
	var reason string
	if totalHoursToAdd > 0 {
		err := addStudentHours(ctx, studentID, totalHoursToAdd, skillType)
		if err != nil {
			return nil, err
		}
		message = fmt.Sprintf("เพิ่ม %d เวลาทำการเช็คอิน/เช็คเอาท์ที่ถูกต้อง", totalHoursToAdd)
		reason = "เช็คอิน/เช็คเอาท์ตรงเวลา - เพิ่มชั่วโมง"
	} else if totalHoursToAdd < 0 {
		err := removeStudentHours(ctx, studentID, -totalHoursToAdd, skillType)
		if err != nil {
			return nil, err
		}
		message = fmt.Sprintf("ลบ %d เวลาทำการเช็คอิน/เช็คเอาท์ที่ไม่เหมาะสม", -totalHoursToAdd)
		reason = "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา - ลบชั่วโมง"
	} else {
		message = "ไม่มีการเปลี่ยนแปลงชั่วโมง - เวลาเช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์"
		reason = "เช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์ - ไม่เปลี่ยนแปลงชั่วโมง"
	}

	// บันทึกประวัติการเปลี่ยนแปลง (ทุกกรณี)
	err = saveHourChangeHistory(ctx, studentID, student.Name, student.Code,
		activityItem.ActivityID.Hex(), "", activityItemID.Hex(), "", skillType,
		totalHoursToAdd, reason)
	if err != nil {
		// ไม่ return error เพราะการบันทึกประวัติไม่ควรทำให้การประมวลผลล้มเหลว
		fmt.Printf("Warning: Failed to save hour change history: %v\n", err)
	}

	return &HourChangeResult{
		StudentID:   studentID.Hex(),
		StudentName: student.Name,
		StudentCode: student.Code,
		SkillType:   skillType,
		HoursChange: totalHoursToAdd,
		Message:     message,
	}, nil
}

// calculateHoursForDate calculates hours to add/remove for a specific date
func calculateHoursForDate(dateInfo models.Dates, checkin *time.Time, checkout *time.Time, activityHour int) int {
	// แปลงเวลาเริ่มเป็น time.Time
	startTime, err := parseTime(dateInfo.Date, dateInfo.Stime)
	if err != nil {
		return 0
	}

	// กำหนดเวลาที่อนุญาตให้ check-in (15 นาทีก่อนเริ่ม)
	allowedCheckinTime := startTime.Add(-15 * time.Minute)

	// เงื่อนไขการเพิ่มหรือลบชั่วโมง
	if checkin != nil && checkout != nil {
		// มีทั้ง check-in และ checkout
		if checkin.Before(allowedCheckinTime) || checkin.Equal(allowedCheckinTime) {
			// Check-in ก่อนหรือเท่ากับเวลาเริ่ม + 15 นาที
			return activityHour
		} else {
			// Check-in หลังเวลาเริ่ม + 15 นาที
			return 0
		}
	} else if checkin != nil && checkout == nil {
		// มี check-in แต่ไม่มี checkout
		return 0
	} else if checkin == nil && checkout != nil {
		// ไม่มี check-in แต่มี checkout
		return 0
	} else {
		// ไม่มีทั้ง check-in และ checkout
		return -activityHour // ลบชั่วโมง
	}
}

// parseTime parses date and time string to time.Time
func parseTime(date, timeStr string) (time.Time, error) {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	return time.ParseInLocation("2006-01-02 15:04", date+" "+timeStr, loc)
}

// addStudentHours adds hours to student's skill count based on activity skill type
func addStudentHours(ctx context.Context, studentID primitive.ObjectID, hours int, skillType string) error {
	// ดึงข้อมูลนักเรียน
	var student models.Student
	err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student)
	if err != nil {
		return fmt.Errorf("student not found: %v", err)
	}

	// อัพเดทชั่วโมงตาม skill type
	var update bson.M
	switch skillType {
	case "soft":
		update = bson.M{
			"$inc": bson.M{
				"softSkill": hours,
			},
		}
	case "hard":
		update = bson.M{
			"$inc": bson.M{
				"hardSkill": hours,
			},
		}
	default:
		return fmt.Errorf("invalid skill type: %s", skillType)
	}

	_, err = DB.StudentCollection.UpdateOne(ctx, bson.M{"_id": studentID}, update)
	if err != nil {
		return fmt.Errorf("ไม่สามารถอัปเดตชั่วโมงเรียนของนักศึกษาได้: %v", err)
	}

	return nil
}

// removeStudentHours removes hours from student's skill count based on activity skill type
func removeStudentHours(ctx context.Context, studentID primitive.ObjectID, hours int, skillType string) error {
	// ดึงข้อมูลนักเรียน
	var student models.Student
	err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student)
	if err != nil {
		return fmt.Errorf("student not found: %v", err)
	}

	// อัพเดทชั่วโมงตาม skill type
	var update bson.M
	switch skillType {
	case "soft":
		// คำนวณชั่วโมงที่จะลบ (ไม่ให้ติดลบ)
		softSkillToRemove := hours
		if student.SoftSkill < hours {
			softSkillToRemove = student.SoftSkill
		}
		update = bson.M{
			"$inc": bson.M{
				"softSkill": -softSkillToRemove,
			},
		}
	case "hard":
		// คำนวณชั่วโมงที่จะลบ (ไม่ให้ติดลบ)
		hardSkillToRemove := hours
		if student.HardSkill < hours {
			hardSkillToRemove = student.HardSkill
		}
		update = bson.M{
			"$inc": bson.M{
				"hardSkill": -hardSkillToRemove,
			},
		}
	default:
		return fmt.Errorf("invalid skill type: %s", skillType)
	}

	_, err = DB.StudentCollection.UpdateOne(ctx, bson.M{"_id": studentID}, update)
	if err != nil {
		return fmt.Errorf("ไม่สามารถอัปเดตชั่วโมงเรียนของนักศึกษาได้: %v", err)
	}

	return nil
}

// saveHourChangeHistory บันทึกประวัติการเปลี่ยนแปลงชั่วโมง
func saveHourChangeHistory(ctx context.Context, studentID primitive.ObjectID, studentName, studentCode string,
	activityID, activityName, activityItemID, activityItemName, skillType string,
	hoursChange int, reason string) error {

	// กำหนด changeType ตาม hoursChange
	var changeType string
	if hoursChange > 0 {
		changeType = "add"
	} else if hoursChange < 0 {
		changeType = "remove"
	} else {
		changeType = "no_change"
	}

	// แปลง activityID และ activityItemID เป็น ObjectID
	activityObjID, err := primitive.ObjectIDFromHex(activityID)
	if err != nil {
		return fmt.Errorf("invalid activity ID format: %v", err)
	}

	activityItemObjID, err := primitive.ObjectIDFromHex(activityItemID)
	if err != nil {
		return fmt.Errorf("invalid activity item ID format: %v", err)
	}

	// สร้างประวัติการเปลี่ยนแปลง
	history := models.HourChangeHistory{
		ID:               primitive.NewObjectID(),
		StudentID:        studentID,
		StudentName:      studentName,
		StudentCode:      studentCode,
		ActivityID:       activityObjID,
		ActivityName:     activityName,
		ActivityItemID:   activityItemObjID,
		ActivityItemName: activityItemName,
		SkillType:        skillType,
		HoursChange:      hoursChange,
		ChangeType:       changeType,
		Reason:           reason,
		ChangedAt:        time.Now(),
	}

	// บันทึกลงฐานข้อมูล
	_, err = DB.HourChangeHistoryCollection.InsertOne(ctx, history)
	if err != nil {
		return fmt.Errorf("ไม่สามารถบันทึกประวัติการเปลี่ยนแปลงชั่วโมงได้: %v", err)
	}

	return nil
}
