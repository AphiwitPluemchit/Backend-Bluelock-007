package checkInOut

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/enrollments"
	hourhistory "Backend-Bluelock-007/src/services/hour-history"
	"context"
	"fmt"
	"log"
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
		_, err := processStudentHours(
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
			programName := deref(program.Name) // ใช้ชื่อ program จริง
			if programName == "" {
				programName = "Unknown Program"
			}
			result.Results = append(result.Results, models.HourChangeHistory{
				ID:           primitive.NewObjectID(),
				StudentID:    en.StudentID,
				EnrollmentID: &en.ID,
				SourceType:   "program",
				SourceID:     programItem.ProgramID,
				SkillType:    program.Skill,
				HourChange:   0,
				Title:        programName, // ใช้ชื่อ program แทน
				Remark:       fmt.Sprintf("Error: %v", err),
				ChangeAt:     time.Now(),
			})
			continue
		}

		result.SuccessCount++
		// h is now nil, no need to append it
		// result.Results = append(result.Results, *h)
	}

	return result, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// findTodayCheckinRecord หา record ของวันที่ระบุที่มี check-in อยู่แล้ว
// คืนค่า index ของ record ที่เจอ หรือ -1 ถ้าไม่เจอ
func findTodayCheckinRecord(records []models.CheckinoutRecord, dateKey string, loc *time.Location) int {
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

// SaveCheckInOut บันทึกการเช็คชื่อเข้า/ออก และอัปเดต participation
func SaveCheckInOut(studentId, programId, checkType string) error {
	ctx := context.TODO()

	// หา programItemId ที่นิสิตลงทะเบียนใน program นี้ (1 enrollment ต่อ 1 program)
	programItemId, found := enrollments.FindEnrolledProgramItem(studentId, programId)
	if !found {
		return fmt.Errorf("คุณไม่ได้ลงทะเบียนกิจกรรมนี้")
	}

	uID, err1 := primitive.ObjectIDFromHex(studentId)
	programItemID, err2 := primitive.ObjectIDFromHex(programItemId)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	now := time.Now()
	loc, _ := time.LoadLocation("Asia/Bangkok")
	dateKey := now.In(loc).Format("2006-01-02")

	// 1) ดึงข้อมูล Enrollment & ProgramItem
	var enrollment models.Enrollment
	if err := DB.EnrollmentCollection.FindOne(ctx,
		bson.M{"studentId": uID, "programItemId": programItemID},
	).Decode(&enrollment); err != nil {
		return fmt.Errorf("ไม่พบการลงทะเบียนของกิจกรรมนี้")
	}

	var programItem models.ProgramItem
	if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": programItemID}).Decode(&programItem); err != nil {
		return fmt.Errorf("ไม่พบข้อมูล program item")
	}

	// 2) ตรวจสอบว่าวันนี้อยู่ในตารางกิจกรรมหรือไม่
	today := now.In(loc).Format("2006-01-02")
	allowed := false
	for _, d := range programItem.Dates {
		if d.Date == today {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("ไม่อนุญาตเช็คชื่อ: วันนี้ (%s) ไม่มีตารางกิจกรรมของรายการนี้", today)
	}

	// 3) เตรียม records
	records := []models.CheckinoutRecord{}
	if enrollment.CheckinoutRecord != nil {
		records = *enrollment.CheckinoutRecord
	}

	// 4) บันทึก Check-in หรือ Check-out
	switch checkType {
	case "checkin":
		// ตรวจสอบว่าเคยเช็คอินวันนี้แล้วหรือไม่
		if idx := findTodayCheckinRecord(records, dateKey, loc); idx >= 0 {
			return fmt.Errorf("คุณได้เช็คชื่อ checkin แล้วในวันนี้")
		}

		// สร้าง record ใหม่สำหรับ check-in วันนี้
		t := now
		records = append(records, models.CheckinoutRecord{
			ID:      primitive.NewObjectID(),
			Checkin: &t,
		})

		// อัปเดต Hour Change History status จาก Upcoming → Participating
		if err := hourhistory.RecordCheckinActivity(ctx, enrollment.ID, dateKey); err != nil {
			log.Printf("⚠️ Warning: failed to record checkin activity: %v", err)
		}

	case "checkout":
		// หา record ของวันนี้ที่มี check-in อยู่แล้ว
		idx := findTodayCheckinRecord(records, dateKey, loc)

		if idx >= 0 {
			// เจอ record ของวันนี้
			if records[idx].Checkout != nil {
				return fmt.Errorf("คุณได้เช็คชื่อ checkout แล้วในวันนี้")
			}
			// อัปเดต checkout
			t := now
			records[idx].Checkout = &t
		} else {
			// ไม่เจอ record ของวันนี้ → สร้างใหม่ (checkout-only case)
			t := now
			records = append(records, models.CheckinoutRecord{
				ID:       primitive.NewObjectID(),
				Checkout: &t,
			})
		}

	default:
		return fmt.Errorf("ประเภทการเช็คชื่อไม่ถูกต้อง")
	}

	// 5) คำนวณ participation สำหรับ record ของวันนี้เท่านั้น
	calculateParticipationForDate(&records, programItem.Dates, dateKey, loc)

	// 6) คำนวณ attendedAllDays
	attendedAll := checkAttendedAllDays(records, programItem.Dates)

	// 7) บันทึกลง Enrollment
	update := bson.M{
		"$set": bson.M{
			"checkinoutRecord": records,
			"attendedAllDays":  attendedAll,
		},
	}
	if _, err := DB.EnrollmentCollection.UpdateOne(
		ctx,
		bson.M{"studentId": uID, "programItemId": programItemID},
		update,
	); err != nil {
		return err
	}

	return nil
}

// calculateParticipationForDate คำนวณสถานะ participation สำหรับ record ของวันที่ระบุเท่านั้น
func calculateParticipationForDate(records *[]models.CheckinoutRecord, dates []models.Dates, dateKey string, loc *time.Location) {
	// หาเวลาเริ่มของวันที่ระบุ
	var startTime *time.Time
	for _, d := range dates {
		if d.Date == dateKey && d.Stime != "" {
			if st, err := time.ParseInLocation("2006-01-02 15:04", d.Date+" "+d.Stime, loc); err == nil {
				startTime = &st
				break
			}
		}
	}

	// หา record ของวันนี้และคำนวณ participation
	for i := range *records {
		var recDateKey string
		if (*records)[i].Checkin != nil {
			recDateKey = (*records)[i].Checkin.In(loc).Format("2006-01-02")
		} else if (*records)[i].Checkout != nil {
			recDateKey = (*records)[i].Checkout.In(loc).Format("2006-01-02")
		}

		// คำนวณเฉพาะ record ของวันที่ระบุ
		if recDateKey != dateKey {
			continue
		}

		participation := "ยังไม่เข้าร่วมกิจกรรม"
		hasIn := (*records)[i].Checkin != nil
		hasOut := (*records)[i].Checkout != nil

		switch {
		case hasIn && hasOut:
			// มีทั้ง checkin และ checkout
			if startTime != nil {
				// อนุญาตเช็คอินก่อนเวลา 30 นาที และหลังเวลา 30 นาที (ไม่ถือว่าสาย)
				earlyLimit := startTime.Add(-30 * time.Minute)
				lateLimit := startTime.Add(30 * time.Minute)

				checkinTime := (*records)[i].Checkin.In(loc)
				if (checkinTime.Equal(earlyLimit) || checkinTime.After(earlyLimit)) &&
					(checkinTime.Before(lateLimit) || checkinTime.Equal(lateLimit)) {
					participation = "เช็คอิน/เช็คเอาท์ตรงเวลา"
				} else if checkinTime.Before(earlyLimit) {
					participation = "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา (เช็คอินเร็วเกินไป)"
				} else {
					participation = "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา (เช็คอินสาย)"
				}
			} else {
				participation = "เช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์ (ไม่พบเวลาเริ่มกิจกรรมของวันนั้น)"
			}

		case hasIn && !hasOut:
			// มี checkin แต่ยังไม่ checkout
			if startTime != nil {
				earlyLimit := startTime.Add(-30 * time.Minute)
				lateLimit := startTime.Add(30 * time.Minute)
				checkinTime := (*records)[i].Checkin.In(loc)

				if (checkinTime.Equal(earlyLimit) || checkinTime.After(earlyLimit)) &&
					(checkinTime.Before(lateLimit) || checkinTime.Equal(lateLimit)) {
					participation = "เช็คอินแล้ว (รอเช็คเอาท์)"
				} else {
					participation = "เช็คอินแล้ว (เวลาไม่เข้าเกณฑ์)"
				}
			} else {
				participation = "เช็คอินแล้ว (ไม่พบเวลาเริ่มกิจกรรม)"
			}

		case !hasIn && hasOut:
			// มี checkout แต่ไม่มี checkin (กรณีลืมเช็คอิน)
			participation = "เช็คเอาท์อย่างเดียว (ข้อมูลไม่ครบ)"
		}

		(*records)[i].Participation = &participation
		break // เจอแล้ว ไม่ต้องหาต่อ
	}
}

// checkAttendedAllDays ตรวจสอบว่านิสิตเข้าร่วมครบทุกวันหรือไม่
func checkAttendedAllDays(records []models.CheckinoutRecord, dates []models.Dates) bool {
	// สร้าง map participation ตามวัน
	participationByDate := make(map[string]string)
	for _, r := range records {
		var dateKey string
		if r.Checkin != nil {
			dateKey = r.Checkin.In(time.UTC).Format("2006-01-02")
		} else if r.Checkout != nil {
			dateKey = r.Checkout.In(time.UTC).Format("2006-01-02")
		}
		if dateKey == "" || r.Participation == nil {
			continue
		}
		participationByDate[dateKey] = *r.Participation
	}

	// ตรวจสอบทุกวันในตาราง
	for _, d := range dates {
		p := participationByDate[d.Date]
		// ถือว่าเข้าร่วมครบถ้า check-in/out ตรงเวลาหรือไม่ตรงเวลา (แต่มีทั้งคู่)
		if !(p == "เช็คอิน/เช็คเอาท์ตรงเวลา" || p == "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา") {
			return false
		}
	}

	return true
}
