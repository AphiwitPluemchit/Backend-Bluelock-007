package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// EnrollmentSummaryResponse โครงสร้างข้อมูล summary ที่ query จาก enrollment โดยตรง
type EnrollmentSummaryResponse struct {
	Registered       int `json:"registered"`       // จำนวนคนลงทะเบียน
	Checkin          int `json:"checkin"`          // เช็คอินตรงเวลา
	CheckinLate      int `json:"checkinLate"`      // เช็คอินสาย
	Checkout         int `json:"checkout"`         // เช็คเอาท์
	NotParticipating int `json:"notParticipating"` // ไม่มา (ลงทะเบียนแต่ไม่เช็คอิน)
}

// GetEnrollmentSummaryByDate ดึงข้อมูล summary จาก enrollment collection โดยตรง
// โดยนับจาก checkinoutRecord ที่มีวันที่ตรงกับ date ที่ส่งมา
func GetEnrollmentSummaryByDate(programID primitive.ObjectID, date string) (*EnrollmentSummaryResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// แปลง date string เป็น time.Time สำหรับการเปรียบเทียบ
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %v", err)
	}

	// สร้างช่วงเวลาของวันนั้น
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// ดึง programItems ของ program นี้
	var programItems []models.ProgramItem
	cursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID})
	if err != nil {
		return nil, fmt.Errorf("failed to find program items: %v", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &programItems); err != nil {
		return nil, fmt.Errorf("failed to decode program items: %v", err)
	}

	if len(programItems) == 0 {
		return &EnrollmentSummaryResponse{
			Registered:       0,
			Checkin:          0,
			CheckinLate:      0,
			Checkout:         0,
			NotParticipating: 0,
		}, nil
	}

	// เก็บ programItemIds เพื่อใช้ query enrollments
	programItemIds := make([]primitive.ObjectID, len(programItems))
	for i, item := range programItems {
		programItemIds[i] = item.ID
	}

	// หา dates ที่ตรงกับ date ที่ระบุ และ checkin time
	var checkInTimeStr string
	found := false

	for _, item := range programItems {
		for _, d := range item.Dates {
			// ถ้าเป็นวันเดียวกับ date ที่ต้องการ
			if d.Date == date {
				if d.Stime != "" {
					checkInTimeStr = d.Stime
					found = true
					break
				}
			}
		}
		if found {
			break
		}
	}

	// ถ้าไม่เจอให้ใช้ค่า default (08:00)
	if checkInTimeStr == "" {
		checkInTimeStr = "08:00"
	}

	// แปลง checkInTimeStr เป็น time.Time
	checkInTime, err := time.Parse("15:04", checkInTimeStr)
	if err != nil {
		checkInTime = startOfDay.Add(8 * time.Hour)
	} else {
		checkInTime = time.Date(
			targetDate.Year(), targetDate.Month(), targetDate.Day(),
			checkInTime.Hour(), checkInTime.Minute(), 0, 0, targetDate.Location(),
		)
	}

	// Query enrollments
	filter := bson.M{"programItemId": bson.M{"$in": programItemIds}}
	enrollmentCursor, err := DB.EnrollmentCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find enrollments: %v", err)
	}
	defer enrollmentCursor.Close(ctx)

	var enrollments []models.Enrollment
	if err = enrollmentCursor.All(ctx, &enrollments); err != nil {
		return nil, fmt.Errorf("failed to decode enrollments: %v", err)
	}

	summary := &EnrollmentSummaryResponse{
		Registered:       len(enrollments),
		Checkin:          0,
		CheckinLate:      0,
		Checkout:         0,
		NotParticipating: 0,
	}

	// นับจำนวนการเช็คอิน/เช็คเอาท์
	for _, enrollment := range enrollments {
		if enrollment.CheckinoutRecord == nil || len(*enrollment.CheckinoutRecord) == 0 {
			continue
		}

		hasCheckinOnDate := false
		hasCheckoutOnDate := false

		for _, record := range *enrollment.CheckinoutRecord {
			// ตรวจสอบว่า record นี้อยู่ในวันที่ที่ต้องการหรือไม่
			if record.Checkin != nil && record.Checkin.After(startOfDay) && record.Checkin.Before(endOfDay) {
				hasCheckinOnDate = true

				// ตรวจสอบว่าเช็คอินตรงเวลาหรือสาย
				// สมมติว่าสายถ้าเช็คอินหลังจาก checkInTime + 15 นาที
				lateThreshold := time.Date(
					targetDate.Year(), targetDate.Month(), targetDate.Day(),
					checkInTime.Hour(), checkInTime.Minute(), checkInTime.Second(),
					0, targetDate.Location(),
				).Add(15 * time.Minute)

				if record.Checkin.After(lateThreshold) {
					summary.CheckinLate++
				} else {
					summary.Checkin++
				}
			}

			// ตรวจสอบ checkout
			if record.Checkout != nil && record.Checkout.After(startOfDay) && record.Checkout.Before(endOfDay) {
				hasCheckoutOnDate = true
				summary.Checkout++
			}
		}

		// ถ้าไม่มีการเช็คอินเลยในวันนั้น = ไม่มา
		if !hasCheckinOnDate && !hasCheckoutOnDate {
			// ไม่นับเพิ่มตัวแปรใด ๆ จะคำนวณจาก registered - (checkin + checkinLate)
		}
	}

	// คำนวณ NotParticipating = ลงทะเบียน - (เช็คอิน + เช็คอินสาย)
	summary.NotParticipating = summary.Registered - (summary.Checkin + summary.CheckinLate)
	if summary.NotParticipating < 0 {
		summary.NotParticipating = 0
	}

	return summary, nil
}

// GetEnrollmentSummaryByDateV2 ดึงข้อมูล summary จาก enrollment collection โดยใช้ aggregation pipeline
// เพื่อประสิทธิภาพที่ดีกว่าในกรณีที่มีข้อมูลมาก
func GetEnrollmentSummaryByDateV2(programID primitive.ObjectID, date string) (*EnrollmentSummaryResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// แปลง date string เป็น time.Time
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %v", err)
	}

	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// ดึง programItems เพื่อหา late threshold
	var programItems []models.ProgramItem
	cursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID})
	if err != nil {
		return nil, fmt.Errorf("failed to find program items: %v", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &programItems); err != nil {
		return nil, fmt.Errorf("failed to decode program items: %v", err)
	}

	if len(programItems) == 0 {
		return &EnrollmentSummaryResponse{}, nil
	}

	programItemIds := make([]primitive.ObjectID, len(programItems))
	for i, item := range programItems {
		programItemIds[i] = item.ID
	}

	// หา checkInTime จาก dates
	var checkInTimeStr string
	foundTime := false
	for _, item := range programItems {
		for _, d := range item.Dates {
			if d.Date == date && d.Stime != "" {
				checkInTimeStr = d.Stime
				foundTime = true
				break
			}
		}
		if foundTime {
			break
		}
	}

	// ถ้าไม่เจอให้ใช้ค่า default
	var checkInTime time.Time
	if checkInTimeStr == "" {
		checkInTime = startOfDay.Add(8 * time.Hour)
	} else {
		parsedTime, err := time.Parse("15:04", checkInTimeStr)
		if err != nil {
			checkInTime = startOfDay.Add(8 * time.Hour)
		} else {
			checkInTime = time.Date(
				targetDate.Year(), targetDate.Month(), targetDate.Day(),
				parsedTime.Hour(), parsedTime.Minute(), 0, 0, targetDate.Location(),
			)
		}
	}

	lateThreshold := time.Date(
		targetDate.Year(), targetDate.Month(), targetDate.Day(),
		checkInTime.Hour(), checkInTime.Minute(), checkInTime.Second(),
		0, targetDate.Location(),
	).Add(15 * time.Minute)

	// Aggregation Pipeline
	pipeline := mongo.Pipeline{
		// Match enrollments ของ program
		{{Key: "$match", Value: bson.M{
			"programItemId": bson.M{"$in": programItemIds},
		}}},

		// Unwind checkinoutRecord
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$checkinoutRecord",
			"preserveNullAndEmptyArrays": true,
		}}},

		// Project เพื่อตรวจสอบเงื่อนไข
		{{Key: "$project", Value: bson.M{
			"_id":           1,
			"studentId":     1,
			"programItemId": 1,
			"hasCheckin": bson.M{
				"$cond": bson.M{
					"if": bson.M{
						"$and": []interface{}{
							bson.M{"$ne": []interface{}{"$checkinoutRecord.checkin", nil}},
							bson.M{"$gte": []interface{}{"$checkinoutRecord.checkin", startOfDay}},
							bson.M{"$lt": []interface{}{"$checkinoutRecord.checkin", endOfDay}},
						},
					},
					"then": 1,
					"else": 0,
				},
			},
			"hasCheckinLate": bson.M{
				"$cond": bson.M{
					"if": bson.M{
						"$and": []interface{}{
							bson.M{"$ne": []interface{}{"$checkinoutRecord.checkin", nil}},
							bson.M{"$gte": []interface{}{"$checkinoutRecord.checkin", startOfDay}},
							bson.M{"$lt": []interface{}{"$checkinoutRecord.checkin", endOfDay}},
							bson.M{"$gt": []interface{}{"$checkinoutRecord.checkin", lateThreshold}},
						},
					},
					"then": 1,
					"else": 0,
				},
			},
			"hasCheckinOnTime": bson.M{
				"$cond": bson.M{
					"if": bson.M{
						"$and": []interface{}{
							bson.M{"$ne": []interface{}{"$checkinoutRecord.checkin", nil}},
							bson.M{"$gte": []interface{}{"$checkinoutRecord.checkin", startOfDay}},
							bson.M{"$lt": []interface{}{"$checkinoutRecord.checkin", endOfDay}},
							bson.M{"$lte": []interface{}{"$checkinoutRecord.checkin", lateThreshold}},
						},
					},
					"then": 1,
					"else": 0,
				},
			},
			"hasCheckout": bson.M{
				"$cond": bson.M{
					"if": bson.M{
						"$and": []interface{}{
							bson.M{"$ne": []interface{}{"$checkinoutRecord.checkout", nil}},
							bson.M{"$gte": []interface{}{"$checkinoutRecord.checkout", startOfDay}},
							bson.M{"$lt": []interface{}{"$checkinoutRecord.checkout", endOfDay}},
						},
					},
					"then": 1,
					"else": 0,
				},
			},
		}}},

		// Group by studentId และนับ
		{{Key: "$group", Value: bson.M{
			"_id":              "$studentId",
			"hasCheckin":       bson.M{"$max": "$hasCheckin"},
			"hasCheckinLate":   bson.M{"$max": "$hasCheckinLate"},
			"hasCheckinOnTime": bson.M{"$max": "$hasCheckinOnTime"},
			"hasCheckout":      bson.M{"$max": "$hasCheckout"},
		}}},

		// Group รวมทั้งหมด
		{{Key: "$group", Value: bson.M{
			"_id":         nil,
			"registered":  bson.M{"$sum": 1},
			"checkin":     bson.M{"$sum": "$hasCheckinOnTime"},
			"checkinLate": bson.M{"$sum": "$hasCheckinLate"},
			"checkout":    bson.M{"$sum": "$hasCheckout"},
		}}},
	}

	aggCursor, err := DB.EnrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate enrollments: %v", err)
	}
	defer aggCursor.Close(ctx)

	var results []bson.M
	if err = aggCursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode aggregation results: %v", err)
	}

	summary := &EnrollmentSummaryResponse{
		Registered:       0,
		Checkin:          0,
		CheckinLate:      0,
		Checkout:         0,
		NotParticipating: 0,
	}

	if len(results) > 0 {
		result := results[0]
		if val, ok := result["registered"].(int32); ok {
			summary.Registered = int(val)
		}
		if val, ok := result["checkin"].(int32); ok {
			summary.Checkin = int(val)
		}
		if val, ok := result["checkinLate"].(int32); ok {
			summary.CheckinLate = int(val)
		}
		if val, ok := result["checkout"].(int32); ok {
			summary.Checkout = int(val)
		}

		summary.NotParticipating = summary.Registered - (summary.Checkin + summary.CheckinLate)
		if summary.NotParticipating < 0 {
			summary.NotParticipating = 0
		}
	}

	return summary, nil
}
