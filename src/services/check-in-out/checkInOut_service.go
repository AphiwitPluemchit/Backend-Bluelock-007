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

// GetCheckinStatus returns all check-in/out records for a student and programItemId
func GetCheckinStatus(studentId, programItemId string) ([]map[string]interface{}, error) {
	uID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(programItemId)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	filter := bson.M{
		"studentId":     uID,
		"programItemId": aID,
	}

	cursor, err := DB.CheckinCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถค้นหาข้อมูลเช็คชื่อได้")
	}
	defer cursor.Close(context.TODO())

	loc, _ := time.LoadLocation("Asia/Bangkok")

	// แยก checkin/checkout ตามวัน
	type rec struct {
		Type      string    `bson:"type"`
		Timestamp time.Time `bson:"timestamp"`
	}
	var checkins, checkouts []time.Time
	for cursor.Next(context.TODO()) {
		var r rec
		if err := cursor.Decode(&r); err != nil {
			continue
		}
		t := r.Timestamp.In(loc)
		switch r.Type {
		case "checkin":
			checkins = append(checkins, t)
		case "checkout":
			checkouts = append(checkouts, t)
		}
	}

	// จับคู่ checkin/checkout ตามลำดับเวลา
	var results []map[string]interface{}
	usedCheckout := make([]bool, len(checkouts))
	for _, ci := range checkins {
		// หา checkout ที่เร็วที่สุดหลัง checkin นี้
		var co *time.Time
		for i, c := range checkouts {
			if !usedCheckout[i] && c.After(ci) {
				co = &c
				usedCheckout[i] = true
				break
			}
		}
		result := map[string]interface{}{
			"checkin": ci,
		}
		if co != nil {
			result["checkout"] = *co
		}
		results = append(results, result)
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
	// หาวันนี้ (ตัดเวลา)
	now := time.Now()
	y, m, d := now.Date()
	loc := now.Location()
	startOfDay := time.Date(y, m, d, 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)
	// เช็คว่ามี record ซ้ำในวันเดียวกันหรือยัง
	filter := bson.M{
		"studentId":     uID,
		"programItemId": aID,
		"type":          checkType,
		"timestamp": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	}
	count, err := DB.CheckinCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("คุณได้เช็คชื่อ %s แล้วในวันนี้", checkType)
	}
	// Insert ใหม่
	checkinRecord := models.CheckinRecord{
		StudentID:     uID,
		ProgramItemID: aID,
		Type:          checkType,
		Timestamp:     now,
	}

	_, err = DB.CheckinCollection.InsertOne(context.TODO(), checkinRecord)
	if err != nil {
		return err
	}

	// อัปเดตข้อมูลการเช็คชื่อในเอกสาร Enrollment ให้สะท้อนสถานะล่าสุด
	// คำนวณคู่ checkin/checkout ใหม่ แล้วเซ็ตลง field checkinoutRecord ของ enrollment
	if status, _ := enrollments.GetCheckinStatus(userId, programItemId); status != nil {
		update := bson.M{"$set": bson.M{"checkinoutRecord": status}}
		_, _ = DB.EnrollmentCollection.UpdateOne(
			context.TODO(),
			bson.M{"studentId": uID, "programItemId": aID},
			update,
		)
	}

	// หากเป็นการ checkout ให้รีเฟรช participation ภายในอาเรย์ checkinoutRecord เท่านั้น (ไม่แตะฟิลด์อื่น)
	if checkType == "checkout" {
		if status, _ := enrollments.GetCheckinStatus(userId, programItemId); status != nil {
			_, _ = DB.EnrollmentCollection.UpdateOne(
				context.TODO(),
				bson.M{"studentId": uID, "programItemId": aID},
				bson.M{"$set": bson.M{"checkinoutRecord": status}},
			)

			// เพิ่มการตรวจสอบว่าเข้าร่วมครบทุกวันที่กำหนดหรือไม่
			// เกณฑ์: วันนั้นถือว่า "เข้าร่วม" หาก participation เป็น "เช็คอิน/เช็คเอาท์ตรงเวลา" หรือ "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา"
			ctx := context.TODO()
			var programItem models.ProgramItem
			if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": aID}).Decode(&programItem); err == nil {
				loc, _ := time.LoadLocation("Asia/Bangkok")
				// map วันที่ -> participation
				participationByDate := make(map[string]string)
				for _, r := range status {
					var dateKey string
					if r.Checkin != nil {
						dateKey = r.Checkin.In(loc).Format("2006-01-02")
					} else if r.Checkout != nil {
						dateKey = r.Checkout.In(loc).Format("2006-01-02")
					}
					if dateKey == "" || r.Participation == nil {
						continue
					}
					participationByDate[dateKey] = *r.Participation
				}

				attendedAll := true
				for _, d := range programItem.Dates {
					p := participationByDate[d.Date]
					if !(p == "เช็คอิน/เช็คเอาท์ตรงเวลา" || p == "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา") {
						attendedAll = false
						break
					}
				}

				// อัปเดตธง attendedAllDays ใน enrollment (เพิ่มฟิลด์ใหม่นี้ในเอกสาร)
				_, _ = DB.EnrollmentCollection.UpdateOne(
					ctx,
					bson.M{"studentId": uID, "programItemId": aID},
					bson.M{"$set": bson.M{"attendedAllDays": attendedAll}},
				)
			}
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
