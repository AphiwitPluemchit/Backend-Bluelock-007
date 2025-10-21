package checkInOut

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	hourhistory "Backend-Bluelock-007/src/services/hour-history"
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// processStudentHours processes hours for a single student and returns the saved history record
func processStudentHours(
	ctx context.Context,
	enrollmentID primitive.ObjectID, // ✅ เพิ่ม
	studentID primitive.ObjectID,
	programItemID primitive.ObjectID,
	programItem models.ProgramItem,
	skillType string,
) (*models.HourChangeHistory, error) {

	// 1) ดึงข้อมูล Program เพื่อเอาชื่อไปใช้เป็น title
	var program models.Program
	if err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programItem.ProgramID}).Decode(&program); err != nil {
		return nil, fmt.Errorf("program not found: %v", err)
	}

	programName := "Unknown Program"
	if program.Name != nil {
		programName = *program.Name
	}

	// 2) Student
	var student models.Student
	if err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student); err != nil {
		return nil, fmt.Errorf("student not found: %v", err)
	}

	// 3) ใช้ข้อมูลจาก Enrollment และ participation ต่อวัน แทนการอ่าน CheckinCollection ตรงๆ
	var enrollment models.Enrollment
	if err := DB.EnrollmentCollection.FindOne(ctx, bson.M{"_id": enrollmentID}).Decode(&enrollment); err != nil {
		// ถ้า enrollment ไม่พบ ให้ถือว่าไม่มีการเข้าร่วม
		if err := removeStudentHours(ctx, studentID, *programItem.Hour, skillType); err != nil {
			return nil, err
		}
		// บันทึกประวัติ - ใช้ชื่อ program เป็น title
		remark := "ไม่พบ Enrollment สำหรับการคำนวณชั่วโมง"
		_ = hourhistory.SaveHourHistory(
			ctx,
			studentID,
			skillType,
			-*programItem.Hour,
			programName, // ใช้ชื่อ program
			remark,
			"program",
			programItem.ProgramID,
			&enrollmentID,
		)
		return nil, nil
	}

	// 4) map วันที่ -> participation
	loc, _ := time.LoadLocation("Asia/Bangkok")
	participationByDate := map[string]string{}
	if enrollment.CheckinoutRecord != nil {
		for _, rec := range *enrollment.CheckinoutRecord {
			var dateKey string
			if rec.Checkin != nil {
				dateKey = rec.Checkin.In(loc).Format("2006-01-02")
			} else if rec.Checkout != nil {
				dateKey = rec.Checkout.In(loc).Format("2006-01-02")
			}
			if dateKey == "" {
				continue
			}
			if rec.Participation != nil {
				participationByDate[dateKey] = *rec.Participation
			}
		}
	}

	// 5) รวมชั่วโมงจาก participation ต่อวันตามตาราง ProgramItem.Dates
	totalHoursToAdd := 0
	seen := map[string]bool{}
	for _, d := range programItem.Dates {
		day := d.Date
		if seen[day] {
			continue
		}
		seen[day] = true

		part := participationByDate[day]
		switch part {
		case "เช็คอิน/เช็คเอาท์ตรงเวลา":
			totalHoursToAdd += *programItem.Hour
		case "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา":
			// 0 ชั่วโมง
		case "เช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์":
			// 0 ชั่วโมง
		case "ยังไม่เข้าร่วมกิจกรรม":
			totalHoursToAdd += -*programItem.Hour
		default:
			// ไม่มีข้อมูล participation ให้ถือว่าไม่ได้เข้าร่วม
			totalHoursToAdd += -*programItem.Hour
		}
	}

	// 6) Apply hours and save history
	remark := ""

	switch {
	case totalHoursToAdd > 0:
		if err := addStudentHours(ctx, studentID, totalHoursToAdd, skillType); err != nil {
			return nil, err
		}
		remark = "เช็คอิน/เช็คเอาท์เข้าเกณฑ์ - เพิ่มชั่วโมง"

	case totalHoursToAdd < 0:
		if err := removeStudentHours(ctx, studentID, -totalHoursToAdd, skillType); err != nil {
			return nil, err
		}
		remark = "เช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์ - ลบชั่วโมง"

	default:
		remark = "เช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์ - ไม่มีการเปลี่ยนแปลง"
	}

	// บันทึกประวัติ - ใช้ชื่อ program เป็น title
	if err := hourhistory.SaveHourHistory(
		ctx,
		studentID,
		skillType,
		totalHoursToAdd,
		programName, // ใช้ชื่อ program แทน
		remark,
		"program",
		programItem.ProgramID,
		&enrollmentID,
	); err != nil {
		// ไม่ให้ล้ม เพราะการบันทึกประวัติไม่ควร stop flow
		fmt.Printf("Warning: Failed to save hour change history: %v\n", err)
	}

	return nil, nil
}

// calculateHoursForDate calculates hours to add/remove for a specific date
// func calculateHoursForDate(dateInfo models.Dates, checkin *time.Time, checkout *time.Time, programHour int) int {
// 	// แปลงเวลาเริ่มเป็น time.Time
// 	startTime, err := parseTime(dateInfo.Date, dateInfo.Stime)
// 	if err != nil {
// 		return 0
// 	}

// 	// กำหนดเวลาที่อนุญาตให้ check-in (15 นาทีก่อนเริ่ม)
// 	allowedCheckinTime := startTime.Add(-15 * time.Minute)

// 	// เงื่อนไขการเพิ่มหรือลบชั่วโมง
// 	if checkin != nil && checkout != nil {
// 		// มีทั้ง check-in และ checkout
// 		if checkin.Before(allowedCheckinTime) || checkin.Equal(allowedCheckinTime) {
// 			// Check-in ก่อนหรือเท่ากับเวลาเริ่ม + 15 นาที
// 			return programHour
// 		} else {
// 			// Check-in หลังเวลาเริ่ม + 15 นาที
// 			return 0
// 		}
// 	} else if checkin != nil && checkout == nil {
// 		// มี check-in แต่ไม่มี checkout
// 		return 0
// 	} else if checkin == nil && checkout != nil {
// 		// ไม่มี check-in แต่มี checkout
// 		return 0
// 	} else {
// 		// ไม่มีทั้ง check-in และ checkout
// 		return -programHour // ลบชั่วโมง
// 	}
// }

// parseTime parses date and time string to time.Time
func parseTime(date, timeStr string) (time.Time, error) {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	return time.ParseInLocation("2006-01-02 15:04", date+" "+timeStr, loc)
}

// addStudentHours adds hours to student's skill count based on program skill type
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

// removeStudentHours removes hours from student's skill count based on program skill type
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

const (
	ChangeTypeAdd      = "add"
	ChangeTypeRemove   = "remove"
	ChangeTypeNoChange = "no_change"

	RecordTypeProgram     = "program"
	RecordTypeCertificate = "certificate"
)

func ClearToken(programId primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ✅ ลบประวัติการเปลี่ยนแปลงชั่วโมงที่เกี่ยวข้องกับ enrollment นี้
	_, err := DB.QrTokenCollection.DeleteMany(ctx, bson.M{"programId": programId})
	if err != nil {
		log.Printf("⚠️ Warning: Failed to delete QrToken for programId %s: %v", programId.Hex(), err)
		// Don't return error - we don't want to fail unenrollment if history deletion fails
	}
	_, err = DB.QrClaimCollection.DeleteMany(ctx, bson.M{"programId": programId})
	if err != nil {
		log.Printf("⚠️ Warning: Failed to delete QrClaim for programId %s: %v", programId.Hex(), err)
		// Don't return error - we don't want to fail unenrollment if history deletion fails
	}
	return nil
}
