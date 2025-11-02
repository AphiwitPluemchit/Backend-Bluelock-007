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

// convertToObjectID แปลง hex string เป็น ObjectID
func convertToObjectID(id string) (primitive.ObjectID, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.ObjectID{}, fmt.Errorf("รหัสไม่ถูกต้อง")
	}
	return objID, nil
}

// getBangkokTime ดึงเวลาปัจจุบันตาม timezone Bangkok
func getBangkokTime() time.Time {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	return time.Now().In(loc)
}

// getTodayDateKey ดึงวันที่ปัจจุบันในรูปแบบ YYYY-MM-DD
func getTodayDateKey() string {
	return getBangkokTime().Format("2006-01-02")
}

// findEnrollment ค้นหา Enrollment โดย studentId และ programItemId
func findEnrollment(ctx context.Context, studentId, programItemId primitive.ObjectID) (*models.Enrollment, error) {
	var enrollment models.Enrollment
	err := DB.EnrollmentCollection.FindOne(ctx, bson.M{
		"studentId":     studentId,
		"programItemId": programItemId,
	}).Decode(&enrollment)

	if err != nil {
		return nil, fmt.Errorf("ไม่พบการลงทะเบียนของกิจกรรมนี้")
	}
	return &enrollment, nil
}

// findProgramItem ค้นหา ProgramItem โดย programItemId
func findProgramItem(ctx context.Context, programItemId primitive.ObjectID) (*models.ProgramItem, error) {
	var programItem models.ProgramItem
	err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": programItemId}).Decode(&programItem)
	if err != nil {
		return nil, fmt.Errorf("ไม่พบข้อมูล program item")
	}
	return &programItem, nil
}

// isDateAllowed ตรวจสอบว่าวันนี้อยู่ในตารางกิจกรรมหรือไม่
func isDateAllowed(programItem *models.ProgramItem, dateKey string) bool {
	for _, d := range programItem.Dates {
		if d.Date == dateKey {
			return true
		}
	}
	return false
}

// findTodayCheckinRecord หา record ของวันที่ระบุที่มี check-in อยู่แล้ว
func findTodayCheckinRecord(records []models.CheckinoutRecord, dateKey string) int {
	loc, _ := time.LoadLocation("Asia/Bangkok")
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

// checkAttendedAllDays ตรวจสอบว่านิสิตเข้าร่วมครบทุกวันหรือไม่
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

// deref แปลง pointer string เป็น string
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
