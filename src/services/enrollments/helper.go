package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetCheckinStatus(studentId, programItemId string) ([]models.CheckinoutRecord, error) {
	uID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(programItemId)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var enrollment models.Enrollment
	if err := DB.EnrollmentCollection.FindOne(ctx, bson.M{"studentId": uID, "programItemId": aID}).Decode(&enrollment); err != nil {
		return []models.CheckinoutRecord{}, nil
	}
	if enrollment.CheckinoutRecord == nil {
		return []models.CheckinoutRecord{}, nil
	}
	return *enrollment.CheckinoutRecord, nil
}

// FindEnrolledItems คืน programItemIds ทั้งหมดที่นิสิตลงทะเบียนไว้ใน programId นี้
func FindEnrolledItems(userId string, programId string) ([]string, bool) {
	uID, _ := primitive.ObjectIDFromHex(userId)
	aID, _ := primitive.ObjectIDFromHex(programId)

	var enrolledItemIDs []string

	// 1. ดึง enrollments ทั้งหมดของ userId
	cursor, err := DB.EnrollmentCollection.Find(context.TODO(), bson.M{
		"studentId": uID, // หรือ "userId" ถ้าคุณใช้ชื่อนี้
	})
	if err != nil {
		return nil, false
	}
	defer cursor.Close(context.TODO())

	// 2. เช็กแต่ละรายการว่า programItemId → programId ตรงหรือไม่
	for cursor.Next(context.TODO()) {
		var enrollment models.Enrollment
		if err := cursor.Decode(&enrollment); err != nil {
			continue
		}

		var item models.ProgramItem
		err := DB.ProgramItemCollection.FindOne(context.TODO(), bson.M{
			"_id": enrollment.ProgramItemID,
		}).Decode(&item)
		if err == nil && item.ProgramID == aID {
			enrolledItemIDs = append(enrolledItemIDs, enrollment.ProgramItemID.Hex())
		}
	}

	if len(enrolledItemIDs) == 0 {
		return nil, false
	}
	return enrolledItemIDs, true
}

func IsStudentEnrolled(studentId string, programItemId string) bool {
	sID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(programItemId)

	if err1 != nil || err2 != nil {
		log.Printf("Invalid ObjectID: studentId=%s, programItemId=%s", studentId, programItemId)
		return false
	}

	filter := bson.M{
		"studentId":     sID,
		"programItemId": aID,
	}

	count, err := DB.EnrollmentCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		log.Printf("MongoDB error when checking enrollment: %v", err)
		return false
	}

	return count > 0
}

// FindEnrolledProgramItem คืน programItemId ที่นิสิตลงทะเบียนไว้ใน programId นี้
// เนื่องจาก 1 student มี 1 enrollment ต่อ 1 program ใช้ aggregation pipeline ค้นหาตรง ๆ
func FindEnrolledProgramItem(studentID string, programId string) (string, bool) {
	sID, err := primitive.ObjectIDFromHex(studentID)
	if err != nil {
		return "", false
	}
	pID, err := primitive.ObjectIDFromHex(programId)
	if err != nil {
		return "", false
	}

	// ใช้ aggregation pipeline เพื่อ join enrollment กับ program_items
	pipeline := []bson.M{
		// 1. หา enrollment ของ student
		{"$match": bson.M{"studentId": sID}},
		{"$match": bson.M{"programId": pID}},
		// 2. lookup programItem
		{
			"$lookup": bson.M{
				"from":         "program_items",
				"localField":   "programItemId",
				"foreignField": "_id",
				"as":           "programItem",
			},
		},
		// 6. limit 1 (เพราะควรมีแค่ 1 enrollment)
		{"$limit": 1},
	}

	cursor, err := DB.EnrollmentCollection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return "", false
	}
	defer cursor.Close(context.TODO())

	if cursor.Next(context.TODO()) {
		var result struct {
			ProgramItemID primitive.ObjectID `bson:"programItemId"`
		}
		if err := cursor.Decode(&result); err != nil {
			return "", false
		}
		return result.ProgramItemID.Hex(), true
	}

	// log
	log.Printf("No enrollment found for studentId=%s in programId=%s", studentID, programId)

	return "", false
}

func isTimeOverlap(start1, end1, start2, end2 string) bool {
	// ตัวอย่าง: 09:00 < 10:00 -> true (มีเวลาทับซ้อน)
	return !(end1 <= start2 || end2 <= start1)
}

func bangkok() *time.Location {
	loc, _ := time.LoadLocation(tzBangkok)
	return loc
}

const (
	tzBangkok = "Asia/Bangkok"
	fmtDay    = "2006-01-02"
	// fmtISOOffset = "2006-01-02T15:04:05-0700"

	// mongoFmtDay       = "%Y-%m-%d"
	mongoFmtISOOffset = "%Y-%m-%dT%H:%M:%S%z" // จะได้ +0700 (ไม่มี :)
)
const ()

// คืนเวลาเริ่มกิจกรรมของ "วันนั้น" (ถ้าเจอ) จาก ProgramItem.Dates
// stime เป็นรูป "HH:mm"
func startTimeForDate(item *models.ProgramItem, date string, loc *time.Location) (time.Time, bool) {
	for _, d := range item.Dates {
		if d.Date == date && d.Stime != "" {
			if st, err := time.ParseInLocation(fmtDay+" 15:04", d.Date+" "+d.Stime, loc); err == nil {
				return st, true
			}
		}
	}
	return time.Time{}, false
}

// เช็คว่าสายไหม: true ถ้า checkin > start+15m
// ถ้า "ไม่พบเวลาเริ่ม" — ผมเลือกตีเป็น 'สาย' เพื่อให้ Summary แยกออกจากตรงเวลา
func isLateCheckin(item *models.ProgramItem, t time.Time, loc *time.Location) bool {
	day := t.In(loc).Format(fmtDay)
	if st, ok := startTimeForDate(item, day, loc); ok {
		return t.After(st.Add(15 * time.Minute))
	}
	// ไม่พบเวลาเริ่มของวันนั้น: นับเป็น late
	return true
}

func participationFor(item *models.ProgramItem, cin, cout *time.Time, loc *time.Location) *string {
	day := ""
	if cin != nil {
		day = cin.In(loc).Format(fmtDay)
	}
	if day == "" && cout != nil {
		day = cout.In(loc).Format(fmtDay)
	}

	st, ok := startTimeForDate(item, day, loc)
	if !ok {
		var label string
		switch {
		case cin != nil && cout == nil:
			label = "เช็คอินแล้ว (เวลาไม่เข้าเกณฑ์)"
		case cin == nil && cout != nil:
			label = "เช็คเอาท์อย่างเดียว (ข้อมูลไม่ครบ)"
		case cin != nil && cout != nil:
			label = "เช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์ (ไม่พบเวลาเริ่ม)"
		}
		if label != "" {
			log.Printf("[participationFor] noStartTime day=%s cin=%v cout=%v -> participation=%q", day, cin, cout, label)
			return &label
		}
		log.Printf("[participationFor] noStartTime day=%s cin=%v cout=%v -> participation=nil", day, cin, cout)
		return nil
	}

	early := st.Add(-15 * time.Minute)
	late := st.Add(15 * time.Minute)
	onTime := func(t time.Time) bool {
		return (t.Equal(early) || t.After(early)) && (t.Before(late) || t.Equal(late))
	}

	var label string
	switch {
	case cin != nil && cout != nil:
		if onTime(cin.In(loc)) && cout.In(loc).After(st) {
			label = "เช็คอิน/เช็คเอาท์ตรงเวลา"
		} else {
			label = "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา"
		}
	case cin != nil && cout == nil:
		if onTime(cin.In(loc)) {
			label = "เช็คอินแล้ว (รอเช็คเอาท์)"
		} else {
			label = "เช็คอินแล้ว (เวลาไม่เข้าเกณฑ์)"
		}
	case cin == nil && cout != nil:
		label = "เช็คเอาท์อย่างเดียว (ข้อมูลไม่ครบ)"
	default:
		log.Printf("[participationFor] st=%v day=%s cin=%v cout=%v -> participation=nil", st, day, cin, cout)
		return nil
	}

	log.Printf("[participationFor] st=%v early=%v late=%v day=%s cin=%v cout=%v -> participation=%q",
		st, early, late, day, cin, cout, label)
	return &label
}

func dateExistsInItem(item *models.ProgramItem, day string) bool {
	for _, d := range item.Dates {
		if d.Date == day {
			return true
		}
	}
	return false
}
