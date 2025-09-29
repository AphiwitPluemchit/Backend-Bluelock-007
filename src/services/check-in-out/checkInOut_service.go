package checkInOut

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/enrollments"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetCheckinStatus returns all check-in/out records for a student and programItemId from Enrollment
func GetCheckinStatus(studentId, programItemId string) ([]map[string]interface{}, error) {
	uID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(programItemId)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	// อ่านจาก Enrollment.checkinoutRecord เท่านั้น
	var enrollment models.Enrollment
	err := DB.EnrollmentCollection.FindOne(
		context.TODO(),
		bson.M{"studentId": uID, "programItemId": aID},
	).Decode(&enrollment)
	if err != nil {
		// ไม่พบ enrollment ให้คืนค่า array ว่าง
		return []map[string]interface{}{}, nil
	}

	results := []map[string]interface{}{}
	if enrollment.CheckinoutRecord == nil {
		return results, nil
	}
	for _, r := range *enrollment.CheckinoutRecord {
		item := map[string]interface{}{}
		if r.Checkin != nil {
			item["checkin"] = *r.Checkin
		}
		if r.Checkout != nil {
			item["checkout"] = *r.Checkout
		}
		if len(item) > 0 {
			results = append(results, item)
		}
	}
	return results, nil
}

// CreateQRToken creates a new QR token for an programId, valid for 8 seconds
func CreateQRToken(programId string, qrType string) (string, int64, error) {
	token := uuid.NewString()
	programObjID, err := primitive.ObjectIDFromHex(programId)
	if err != nil {
		return "", 0, err
	}
	now := time.Now().Unix()
	expiresAt := now + 30 // 30 วินาที
	qrToken := models.QRToken{
		Token:     token,
		ProgramID: programObjID,
		Type:      qrType,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	_, err = DB.QrTokenCollection.InsertOne(context.TODO(), qrToken)
	if err != nil {
		return "", 0, err
	}
	return token, expiresAt, nil
}

// ClaimQRToken allows a student to claim a QR token if not expired and not already claimed
func ClaimQRToken(token, studentId string) (*models.QRToken, error) {
	ctx := context.TODO()
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return nil, err
	}
	// 1. หาใน qr_claims ก่อน (token+studentId+expireAt>now)
	var claim struct {
		Token     string             `bson:"token"`
		StudentID primitive.ObjectID `bson:"studentId"`
		ProgramID primitive.ObjectID `bson:"programId"`
		Type      string             `bson:"type"`
		ClaimedAt time.Time          `bson:"claimedAt"`
		ExpireAt  time.Time          `bson:"expireAt"`
	}
	err = DB.QrClaimCollection.FindOne(ctx, bson.M{"token": token, "studentId": studentObjID, "expireAt": bson.M{"$gt": time.Now()}}).Decode(&claim)
	if err == nil {
		return &models.QRToken{
			Token:              claim.Token,
			ProgramID:          claim.ProgramID,
			Type:               claim.Type,
			ClaimedByStudentID: &studentObjID,
		}, nil
	}
	// 2. ถ้าไม่เจอ → ไปหาใน qr_tokens (token+expiresAt>now)
	var qrToken models.QRToken
	err = DB.QrTokenCollection.FindOne(ctx, bson.M{"token": token, "expiresAt": bson.M{"$gt": time.Now().Unix()}}).Decode(&qrToken)
	if err != nil {
		return nil, fmt.Errorf("QR token expired or invalid")
	}

	// 3. ตรวจสอบว่านักศึกษาได้ลงทะเบียนในกิจกรรมนี้หรือไม่
	itemIDs, found := enrollments.FindEnrolledItems(studentId, qrToken.ProgramID.Hex())
	if !found || len(itemIDs) == 0 {
		return nil, fmt.Errorf("คุณไม่ได้ลงทะเบียนกิจกรรมนี้")
	}

	// 4. upsert ลง qr_claims (หมดอายุใน 1 ชม. หลัง claim)
	expireAt := time.Now().Add(1 * time.Hour)
	claimDoc := bson.M{
		"token":     token,
		"studentId": studentObjID,
		"programId": qrToken.ProgramID,
		"type":      qrToken.Type,
		"claimedAt": time.Now(),
		"expireAt":  expireAt,
	}
	_, err = DB.QrClaimCollection.UpdateOne(ctx, bson.M{"token": token, "studentId": studentObjID}, bson.M{"$set": claimDoc}, options.Update().SetUpsert(true))
	if err != nil {
		return nil, err
	}
	qrToken.ClaimedByStudentID = &studentObjID
	return &qrToken, nil
}

// ValidateQRToken checks if the token is valid for the student (claimed and not expired)
func ValidateQRToken(token, studentId string) (*models.QRToken, error) {
	ctx := context.TODO()
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
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
	err = DB.QrClaimCollection.FindOne(ctx, bson.M{"token": token, "studentId": studentObjID, "expireAt": bson.M{"$gt": time.Now()}}).Decode(&claim)
	if err != nil {
		return nil, fmt.Errorf("QR token not claimed or expired")
	}

	// ตรวจสอบว่านักศึกษาได้ลงทะเบียนในกิจกรรมนี้หรือไม่
	itemIDs, found := enrollments.FindEnrolledItems(studentId, claim.ProgramID.Hex())
	if !found || len(itemIDs) == 0 {
		return nil, fmt.Errorf("คุณไม่ได้ลงทะเบียนกิจกรรมนี้")
	}

	return &models.QRToken{
		Token:              claim.Token,
		ProgramID:          claim.ProgramID,
		Type:               claim.Type,
		ClaimedByStudentID: &studentObjID,
	}, nil
}

// SaveCheckInOut saves a check-in/out for a specific programItemId, prevents duplicate in the same day
func SaveCheckInOut(userId, programItemId, checkType string) error {
	uID, err1 := primitive.ObjectIDFromHex(userId)
	aID, err2 := primitive.ObjectIDFromHex(programItemId)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Bangkok")
	dateKey := now.In(loc).Format("2006-01-02")

	// ดึง enrollment สำหรับ student+programItem
	var enrollment models.Enrollment
	err := DB.EnrollmentCollection.FindOne(context.TODO(), bson.M{"studentId": uID, "programItemId": aID}).Decode(&enrollment)
	if err != nil {
		return fmt.Errorf("ไม่พบการลงทะเบียนของกิจกรรมนี้")
	}

	// เตรียมอาร์เรย์
	records := []models.CheckinoutRecord{}
	if enrollment.CheckinoutRecord != nil {
		records = append(records, (*enrollment.CheckinoutRecord)...)
	}

	// หาเรคคอร์ดของวันเดียวกันล่าสุด
	var targetIdx int = -1
	for i := len(records) - 1; i >= 0; i-- {
		var d string
		if records[i].Checkin != nil {
			d = records[i].Checkin.In(loc).Format("2006-01-02")
		} else if records[i].Checkout != nil {
			d = records[i].Checkout.In(loc).Format("2006-01-02")
		}
		if d == dateKey {
			targetIdx = i
			break
		}
	}

	switch checkType {
	case "checkin":
		// ป้องกันเช็คอินซ้ำในวันเดียวกัน
		if targetIdx >= 0 && records[targetIdx].Checkin != nil {
			return fmt.Errorf("คุณได้เช็คชื่อ checkin แล้วในวันนี้")
		}
		t := now
		if targetIdx >= 0 {
			records[targetIdx].Checkin = &t
		} else {
			records = append(records, models.CheckinoutRecord{Checkin: &t})
		}
	case "checkout":
		// ต้องมีเรคคอร์ดวันนี้ และยังไม่มี checkout
		if targetIdx >= 0 {
			if records[targetIdx].Checkout != nil {
				return fmt.Errorf("คุณได้เช็คชื่อ checkout แล้วในวันนี้")
			}
			t := now
			records[targetIdx].Checkout = &t
		} else {
			// อนุญาต checkout-only กรณีไม่มี checkin
			t := now
			records = append(records, models.CheckinoutRecord{Checkout: &t})
		}
	default:
		return fmt.Errorf("ประเภทการเช็คชื่อไม่ถูกต้อง")
	}

	// เติมค่า Participation ต่อรายการตามเวลาเริ่มใน ProgramItem
	var programItem models.ProgramItem
	if err := DB.ProgramItemCollection.FindOne(context.TODO(), bson.M{"_id": aID}).Decode(&programItem); err == nil {
		startByDate := make(map[string]time.Time, len(programItem.Dates))
		for _, d := range programItem.Dates {
			if d.Date == "" || d.Stime == "" {
				continue
			}
			if st, err := time.ParseInLocation("2006-01-02 15:04", d.Date+" "+d.Stime, loc); err == nil {
				startByDate[d.Date] = st
			}
		}
		for i := range records {
			var d string
			if records[i].Checkin != nil {
				d = records[i].Checkin.In(loc).Format("2006-01-02")
			} else if records[i].Checkout != nil {
				d = records[i].Checkout.In(loc).Format("2006-01-02")
			}
			participation := "ยังไม่เข้าร่วมกิจกรรม"
			hasIn := records[i].Checkin != nil
			hasOut := records[i].Checkout != nil
			switch {
			case hasIn && hasOut:
				if st, ok := startByDate[d]; ok {
					early := st.Add(-15 * time.Minute)
					late := st.Add(15 * time.Minute)
					if (records[i].Checkin.Equal(early) || records[i].Checkin.After(early)) &&
						(records[i].Checkin.Before(late) || records[i].Checkin.Equal(late)) {
						participation = "เช็คอิน/เช็คเอาท์ตรงเวลา"
					} else {
						participation = "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา"
					}
				} else {
					participation = "เช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์ (ไม่พบเวลาเริ่มกิจกรรมของวันนั้น)"
				}
			case hasIn && !hasOut:
				if st, ok := startByDate[d]; ok && !records[i].Checkin.Before(st.Add(-15*time.Minute)) {
					participation = "เช็คอินแล้ว (รอเช็คเอาท์)"
				} else {
					participation = "เช็คอินแล้ว (เวลาไม่เข้าเกณฑ์)"
				}
			case !hasIn && hasOut:
				participation = "เช็คเอาท์อย่างเดียว (ข้อมูลไม่ครบ)"
			}
			p := participation
			records[i].Participation = &p
		}
		// อัปเดตธง attendedAllDays
		participationByDate := make(map[string]string)
		for _, r := range records {
			var d string
			if r.Checkin != nil {
				d = r.Checkin.In(loc).Format("2006-01-02")
			} else if r.Checkout != nil {
				d = r.Checkout.In(loc).Format("2006-01-02")
			}
			if d == "" || r.Participation == nil {
				continue
			}
			participationByDate[d] = *r.Participation
		}
		attendedAll := true
		for _, d := range programItem.Dates {
			p := participationByDate[d.Date]
			if !(p == "เช็คอิน/เช็คเอาท์ตรงเวลา" || p == "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา") {
				attendedAll = false
				break
			}
		}
		// อัปเดต enrollment ด้วย records และ attendedAllDays
		update := bson.M{"$set": bson.M{"checkinoutRecord": records, "attendedAllDays": attendedAll}}
		_, err = DB.EnrollmentCollection.UpdateOne(context.TODO(), bson.M{"studentId": uID, "programItemId": aID}, update)
		if err != nil {
			return err
		}
	} else {
		// อัปเดตเฉพาะ records ถ้าหา programItem ไม่ได้
		update := bson.M{"$set": bson.M{"checkinoutRecord": records}}
		_, err = DB.EnrollmentCollection.UpdateOne(context.TODO(), bson.M{"studentId": uID, "programItemId": aID}, update)
		if err != nil {
			return err
		}
	}

	return nil
}

// RecordCheckin records a check-in or check-out for a student for all enrolled items in an program
func RecordCheckin(studentId, programItemId, checkType string) error {
	// ดึง programItemIds ทั้งหมดที่นิสิตลงทะเบียนใน program นี้
	itemIDs, found := enrollments.FindEnrolledItems(studentId, programItemId)
	if !found || len(itemIDs) == 0 {
		return fmt.Errorf("คุณไม่ได้ลงทะเบียนกิจกรรมนี้")
	}
	for _, itemID := range itemIDs {
		err := SaveCheckInOut(studentId, itemID, checkType)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetProgramFormId ดึง formId จาก programId
func GetProgramFormId(programId string) (string, error) {
	ctx := context.TODO()
	programObjID, err := primitive.ObjectIDFromHex(programId)
	if err != nil {
		return "", fmt.Errorf("invalid program ID format")
	}

	var program struct {
		FormID primitive.ObjectID `bson:"formId"`
	}

	err = DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programObjID}).Decode(&program)
	if err != nil {
		return "", fmt.Errorf("program not found")
	}

	// ตรวจสอบว่า formId เป็น zero value หรือไม่
	if program.FormID.IsZero() {
		return "", fmt.Errorf("program does not have a form")
	}

	return program.FormID.Hex(), nil
}

type AddHoursForStudentResult struct {
	ProgramItemID string                     `json:"programItemId"`
	ProgramName   string                     `json:"programName"`
	SkillType     string                     `json:"skillType"`
	TotalStudents int                        `json:"totalStudents"`
	SuccessCount  int                        `json:"successCount"`
	ErrorCount    int                        `json:"errorCount"`
	Results       []models.HourChangeHistory `json:"results"`
}

func AddHoursForStudent(programItemId string) (*AddHoursForStudentResult, error) {
	ctx := context.TODO()
	programItemObjID, err := primitive.ObjectIDFromHex(programItemId)
	if err != nil {
		return nil, fmt.Errorf("invalid programItemId format: %v", err)
	}

	// 1) ProgramItem
	var programItem models.ProgramItem
	if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": programItemObjID}).Decode(&programItem); err != nil {
		return nil, fmt.Errorf("program item not found: %v", err)
	}
	if programItem.Hour == nil {
		return nil, fmt.Errorf("program item has no hour value")
	}

	// 2) Program
	var program models.Program
	if err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programItem.ProgramID}).Decode(&program); err != nil {
		return nil, fmt.Errorf("program not found: %v", err)
	}

	// 3) Enrollments
	cur, err := DB.EnrollmentCollection.Find(ctx, bson.M{"programItemId": programItemObjID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch enrollments: %v", err)
	}
	defer cur.Close(ctx)

	var enrollments []models.Enrollment
	if err := cur.All(ctx, &enrollments); err != nil {
		return nil, fmt.Errorf("failed to decode enrollments: %v", err)
	}

	// 4) Result
	result := &AddHoursForStudentResult{
		ProgramItemID: programItemId,
		ProgramName:   deref(program.Name), // เผื่อ program.Name เป็น *string
		SkillType:     program.Skill,
		TotalStudents: len(enrollments),
		Results:       make([]models.HourChangeHistory, 0, len(enrollments)),
	}

	// 5) Process each enrollment
	for _, en := range enrollments {
		h, err := processStudentHours(
			ctx,
			en.ID, // ส่ง enrollmentId เข้าไป
			en.StudentID,
			programItemObjID,
			programItem,
			program.Skill,
		)
		if err != nil {
			result.ErrorCount++
			// กรณี error: แนบข้อมูลเท่าที่รู้ (ไว้โชว์ใน response)
			// ดึง Student เพื่อเติม studentCode กรณี model Enrollment ไม่มี field นี้
			var st models.Student
			_ = DB.StudentCollection.FindOne(ctx, bson.M{"_id": en.StudentID}).Decode(&st)

			result.Results = append(result.Results, models.HourChangeHistory{
				StudentID:     en.StudentID,
				StudentCode:   st.Code,
				ProgramID:     programItem.ProgramID,
				ProgramItemID: programItemObjID,
				EnrollmentID:  &en.ID,
				Type:          RecordTypeProgram,
				SkillType:     program.Skill,
				HoursChange:   0,
				ChangeType:    ChangeTypeNoChange,
				Remark:        fmt.Sprintf("Error: %v", err),
				ChangedAt:     time.Now(),
			})
			continue
		}

		result.SuccessCount++
		result.Results = append(result.Results, *h)
	}

	return result, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// GetHourChangeHistory ดึงประวัติการเปลี่ยนแปลงชั่วโมงของนักเรียน
func GetHourChangeHistory(studentID string, limit int) ([]models.HourChangeHistory, error) {
	ctx := context.TODO()

	studentObjID, err := primitive.ObjectIDFromHex(studentID)
	if err != nil {
		return nil, fmt.Errorf("invalid student ID format: %v", err)
	}

	// สร้าง filter และ options
	filter := bson.M{"studentId": studentObjID}
	opts := options.Find().SetSort(bson.D{{Key: "changedAt", Value: -1}})

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	// ดึงข้อมูล
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

// GetHourChangeHistoryByProgram ดึงประวัติการเปลี่ยนแปลงชั่วโมงของกิจกรรม
func GetHourChangeHistoryByProgram(programID string, limit int) ([]models.HourChangeHistory, error) {
	ctx := context.TODO()

	programObjID, err := primitive.ObjectIDFromHex(programID)
	if err != nil {
		return nil, fmt.Errorf("invalid program ID format: %v", err)
	}

	// สร้าง filter และ options
	filter := bson.M{"programId": programObjID}
	opts := options.Find().SetSort(bson.D{{Key: "changedAt", Value: -1}})

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	// ดึงข้อมูล
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

// GetHourChangeHistorySummary สรุปประวัติการเปลี่ยนแปลงชั่วโมง
func GetHourChangeHistorySummary(studentID string) (map[string]interface{}, error) {
	ctx := context.TODO()

	studentObjID, err := primitive.ObjectIDFromHex(studentID)
	if err != nil {
		return nil, fmt.Errorf("invalid student ID format: %v", err)
	}

	// Pipeline สำหรับ aggregation
	pipeline := []bson.M{
		{"$match": bson.M{"studentId": studentObjID}},
		{"$group": bson.M{
			"_id":        "$changeType",
			"count":      bson.M{"$sum": 1},
			"totalHours": bson.M{"$sum": "$hoursChange"},
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

	// สร้าง summary
	summary := map[string]interface{}{
		"totalRecords": 0,
		"totalAdded":   0,
		"totalRemoved": 0,
		"noChange":     0,
	}

	for _, result := range results {
		changeType := result["_id"].(string)
		count := result["count"].(int32)
		totalHours := result["totalHours"].(int32)

		summary["totalRecords"] = summary["totalRecords"].(int) + int(count)

		switch changeType {
		case "add":
			summary["totalAdded"] = int(totalHours)
		case "remove":
			summary["totalRemoved"] = int(totalHours)
		case "no_change":
			summary["noChange"] = int(count)
		}
	}

	return summary, nil
}
