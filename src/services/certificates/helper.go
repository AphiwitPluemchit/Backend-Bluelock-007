package services

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/courses"
	hourhistory "Backend-Bluelock-007/src/services/hour-history"
	"Backend-Bluelock-007/src/services/students"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// calculateCurrentCertificateHours คำนวณชั่วโมงรวมจาก certificate ที่ approved แล้ว (sourceType = "certificate")
func calculateCurrentCertificateHours(ctx context.Context, studentID primitive.ObjectID, skillType string) (int, error) {
	// Query: หา HourChangeHistory ที่เป็น certificate และ approved
	filter := bson.M{
		"studentId":  studentID,
		"sourceType": "certificate",
		"status":     models.HCStatusApproved,
		"skillType":  skillType,
	}

	cursor, err := DB.HourChangeHistoryCollection.Find(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to query hour change history: %v", err)
	}
	defer cursor.Close(ctx)

	totalHours := 0
	for cursor.Next(ctx) {
		var record models.HourChangeHistory
		if err := cursor.Decode(&record); err != nil {
			continue
		}
		totalHours += record.HourChange
	}

	return totalHours, nil
}

// getMaxTrainingHours คำนวณชั่วโมงอบรมสูงสุดตามประเภท skill และสาขา
func getMaxTrainingHours(skillType string, major string) int {
	if skillType == "soft" {
		return 15 // soft skill: อบรมไม่เกิน 15 ชั่วโมง (ทุกสาขา)
	}

	// hard skill: ขึ้นอยู่กับสาขา
	majorUpper := strings.ToUpper(major)
	if majorUpper == "SE" || majorUpper == "AAI" {
		return 9 // SE และ AAI: อบรมไม่เกิน 9 ชั่วโมง
	}

	// ITDI, CS และสาขาอื่นๆ
	return 6 // อบรมไม่เกิน 6 ชั่วโมง
}

// calculateHoursToAdd คำนวณชั่วโมงที่สามารถเพิ่มได้จริง (ไม่เกิน max)
func calculateHoursToAdd(courseHour, currentHours, maxHours int, studentCode, skillType string) int {
	hoursToAdd := courseHour

	if currentHours+hoursToAdd > maxHours {
		hoursToAdd = maxHours - currentHours
		if hoursToAdd < 0 {
			hoursToAdd = 0
		}

		if hoursToAdd > 0 {
			fmt.Printf("⚠️ Certificate hours capped: Student %s already has %d/%d %s training hours, adding only %d (original: %d)\n",
				studentCode, currentHours, maxHours, skillType, hoursToAdd, courseHour)
		} else {
			fmt.Printf("⚠️ Student %s has reached max %s training hours (%d/%d), no hours added\n",
				studentCode, skillType, currentHours, maxHours)
		}
	}

	return hoursToAdd
}

// saveOrUpdateHourHistory บันทึกหรืออัพเดท hour history record และบันทึก hourHistoryId กลับไปที่ certificate
// ต้องมี hour history record อยู่แล้ว ไม่งั้น error
func saveOrUpdateHourHistory(ctx context.Context, certificate *models.UploadCertificate, course models.Course, skillType string, hoursToAdd int, status string) error {
	now := time.Now()

	// ตรวจสอบว่า certificate มี hourHistoryId หรือไม่
	if certificate.HourHistoryId == nil {
		return fmt.Errorf("certificate %s does not have hourHistoryId", certificate.ID.Hex())
	}

	// ใช้ hourHistoryId จาก certificate เพื่อหา record
	histFilter := bson.M{
		"_id":        *certificate.HourHistoryId,
		"sourceType": "certificate",
		"sourceId":   certificate.ID,
		"studentId":  certificate.StudentId,
	}

	remark := "อนุมัติใบรับรอง"
	if status == models.HCStatusRejected {
		remark = "ปฏิเสธใบรับรอง"
	}

	histUpdate := bson.M{
		"$set": bson.M{
			"status":     status,
			"hourChange": hoursToAdd,
			"remark":     remark,
			"changeAt":   now,
			"title":      course.Name,
			"skillType":  skillType,
		},
	}

	updateResult, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if err != nil {
		return fmt.Errorf("failed to update hour history: %v", err)
	}

	// ถ้าไม่เจอ record ให้ error
	if updateResult.MatchedCount == 0 {
		return fmt.Errorf("hour history record not found for certificate %s (hourHistoryId: %s)",
			certificate.ID.Hex(), certificate.HourHistoryId.Hex())
	}

	fmt.Printf("📝 Updated hour history for certificate %s (ID: %s, status: %s)\n",
		certificate.ID.Hex(), certificate.HourHistoryId.Hex(), status)

	return nil
}

// updateCertificateHoursRejected อัพเดท student hours และ hour history เมื่อ certificate ถูกปฏิเสธหรือยกเลิก
// ต้องมี hour history record อยู่แล้ว ไม่งั้น error
func updateCertificateHoursRejected(ctx context.Context, certificate *models.UploadCertificate) error {
	// Validation: ตรวจสอบว่า certificate ไม่ซ้ำ
	if certificate.IsDuplicate {
		fmt.Printf("Skipping hours removal for duplicate certificate %s\n", certificate.ID.Hex())
		return nil // ไม่ต้อง error แค่ไม่ลบชั่วโมง
	}

	// ตรวจสอบว่ามี hourHistoryId
	if certificate.HourHistoryId == nil {
		return fmt.Errorf("certificate %s does not have hourHistoryId", certificate.ID.Hex())
	}

	// 1. ดึงข้อมูล course
	course, err := courses.GetCourseByID(certificate.CourseId)
	if err != nil {
		return fmt.Errorf("course not found: %v", err)
	}

	if course.Hour <= 0 {
		fmt.Printf("Warning: Course %s has no hours defined (%d), skipping hours removal\n", course.ID.Hex(), course.Hour)
		return nil // ไม่ error แต่ไม่ลบชั่วโมง
	}

	// 2. ดึงข้อมูล student
	student, err := students.GetStudentById(certificate.StudentId)
	if err != nil {
		return fmt.Errorf("student not found: %v", err)
	}

	// 3. กำหนด skill type
	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	// Log remarks
	fmt.Printf("▶️ Old Remark: %s\n", certificate.Remark)

	// 5. อัพเดท hour history record โดยใช้ hourHistoryId
	remark := "ปฏิเสธใบรับรอง"
	if certificate.Remark != "" {
		remark = certificate.Remark
	}

	fmt.Printf("▶️ New Remark for Hour History: %s\n", remark)

	// ใช้ hourHistoryId จาก certificate
	histFilter := bson.M{
		"_id":        *certificate.HourHistoryId,
		"sourceType": "certificate",
		"sourceId":   certificate.ID,
		"studentId":  certificate.StudentId,
	}

	// ตรวจสอบว่า record เดิมเป็นสถานะอะไร
	var existingHistory models.HourChangeHistory
	err = DB.HourChangeHistoryCollection.FindOne(ctx, histFilter).Decode(&existingHistory)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("hour history record not found for certificate %s (hourHistoryId: %s)",
				certificate.ID.Hex(), certificate.HourHistoryId.Hex())
		}
		return fmt.Errorf("failed to find hour history: %v", err)
	}

	// ตรวจสอบว่าเดิมมีชั่วโมงหรือไม่
	var hourChangeValue int
	if existingHistory.Status == models.HCStatusApproved {
		// เดิมเป็น approved (มีชั่วโมง) -> ตั้งชั่วโมงเป็น 0 แทนการลบ (certificate ไม่มีการหักลบ)
		hourChangeValue = 0
	} else {
		// เดิมเป็น pending (ยังไม่มีชั่วโมง) -> ไม่มีการเปลี่ยนแปลง
		hourChangeValue = 0
	}

	histUpdate := bson.M{
		"$set": bson.M{
			"status":     models.HCStatusRejected,
			"hourChange": hourChangeValue,
			"remark":     remark,
			"changeAt":   time.Now(),
			"title":      course.Name,
			"skillType":  skillType,
		},
	}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if err != nil {
		return fmt.Errorf("failed to update hour history: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("failed to update hour history - record not found (ID: %s)", certificate.HourHistoryId.Hex())
	}

	fmt.Printf("📝 Updated hour history (pending/approved -> rejected) for certificate %s (hourChange: %d, ID: %s)\n",
		certificate.ID.Hex(), hourChangeValue, certificate.HourHistoryId.Hex())

	fmt.Printf("❌ Hours set to 0 (certificate does not use negative hours) from student %s for certificate %s\n",
		student.Code, certificate.ID.Hex())

	// 🔄 Update student status หลังจากมีการเปลี่ยนแปลงชั่วโมง
	if err := hourhistory.UpdateStudentStatus(ctx, certificate.StudentId); err != nil {
		fmt.Printf("⚠️ Warning: Failed to update student status for %s: %v\n", student.Code, err)
		// ไม่ return error เพราะการอัปเดตชั่วโมงสำเร็จแล้ว เหลือแค่ status
	}

	return nil
}

// recordCertificateRejection อัพเดท hour history เมื่อ certificate ถูกปฏิเสธจาก pending
// ไม่มีการเปลี่ยนแปลงชั่วโมงจริง (hourChange = 0) แต่บันทึกเป็นประวัติ
// ต้องมี hour history record อยู่แล้ว ไม่งั้น error
func recordCertificateRejection(ctx context.Context, certificate *models.UploadCertificate, adminRemark string) error {
	// ตรวจสอบว่ามี hourHistoryId
	if certificate.HourHistoryId == nil {
		return fmt.Errorf("certificate %s does not have hourHistoryId", certificate.ID.Hex())
	}

	// ดึงข้อมูล course เพื่อหาประเภท skill
	course, err := courses.GetCourseByID(certificate.CourseId)
	if err != nil {
		return fmt.Errorf("course not found: %v", err)
	}

	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	remark := "ปฏิเสธใบรับรอง"
	if adminRemark != "" {
		remark = adminRemark
	}

	// ใช้ hourHistoryId จาก certificate
	histFilter := bson.M{
		"_id":        *certificate.HourHistoryId,
		"sourceType": "certificate",
		"sourceId":   certificate.ID,
		"studentId":  certificate.StudentId,
	}

	histUpdate := bson.M{
		"$set": bson.M{
			"status":     models.HCStatusRejected,
			"hourChange": 0, // ไม่มีการเปลี่ยนแปลงชั่วโมง
			"remark":     remark,
			"changeAt":   time.Now(),
			"title":      course.Name,
			"skillType":  skillType,
		},
	}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if err != nil {
		return fmt.Errorf("failed to update hour history: %v", err)
	}

	// ถ้าไม่เจอ record ให้ error
	if result.MatchedCount == 0 {
		return fmt.Errorf("hour history record not found for certificate %s (hourHistoryId: %s)",
			certificate.ID.Hex(), certificate.HourHistoryId.Hex())
	}

	fmt.Printf("📝 Updated hour history to rejected for certificate %s (ID: %s)\n",
		certificate.ID.Hex(), certificate.HourHistoryId.Hex())

	return nil
}

// recordCertificatePending อัพเดท hour history เมื่อ certificate กลับไปสถานะ pending
// ไม่มีการเปลี่ยนแปลงชั่วโมงจริง (hourChange = 0) แต่บันทึกเป็นประวัติ
// ต้องมี hour history record อยู่แล้ว ไม่งั้น error
func recordCertificatePending(ctx context.Context, certificate *models.UploadCertificate, adminRemark string) error {
	// ไม่ต้องบันทึกถ้าเป็น duplicate
	if certificate.IsDuplicate {
		return nil
	}

	// ตรวจสอบว่ามี hourHistoryId
	if certificate.HourHistoryId == nil {
		return fmt.Errorf("certificate %s does not have hourHistoryId", certificate.ID.Hex())
	}

	// ดึงข้อมูล course เพื่อหาประเภท skill
	course, err := courses.GetCourseByID(certificate.CourseId)
	if err != nil {
		return fmt.Errorf("course not found: %v", err)
	}

	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	remark := "รอให้เจ้าหน้าที่ตรวจสอบ"
	if adminRemark != "" {
		remark = adminRemark
	}

	// ใช้ hourHistoryId จาก certificate
	histFilter := bson.M{
		"_id":        *certificate.HourHistoryId,
		"sourceType": "certificate",
		"sourceId":   certificate.ID,
		"studentId":  certificate.StudentId,
	}

	histUpdate := bson.M{
		"$set": bson.M{
			"status":     models.HCStatusPending,
			"hourChange": 0, // ไม่มีการเปลี่ยนแปลงชั่วโมง
			"remark":     remark,
			"changeAt":   time.Now(),
			"title":      course.Name,
			"skillType":  skillType,
		},
	}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if err != nil {
		return fmt.Errorf("failed to update hour history: %v", err)
	}

	// ถ้าไม่เจอ record ให้ error
	if result.MatchedCount == 0 {
		return fmt.Errorf("hour history record not found for certificate %s (hourHistoryId: %s)",
			certificate.ID.Hex(), certificate.HourHistoryId.Hex())
	}

	fmt.Printf("📝 Updated hour history to pending for certificate %s (ID: %s)\n",
		certificate.ID.Hex(), certificate.HourHistoryId.Hex())

	return nil
}

// RecordUploadPending is an exported helper that controllers can call to record
// a pending-hour-history entry for a newly created upload certificate.
// Note: CreateUploadCertificate now creates hour history automatically,
// so this function is only needed for legacy or special cases.
func RecordUploadPending(certificate *models.UploadCertificate, remark string) error {
	// ถ้ามี hourHistoryId แล้ว แสดงว่า hour history ถูกสร้างไปแล้ว ไม่ต้องสร้างอีก
	if certificate.HourHistoryId != nil {
		fmt.Printf("Certificate %s already has hourHistoryId: %s, skipping creation\n",
			certificate.ID.Hex(), certificate.HourHistoryId.Hex())
		return nil
	}

	return recordCertificatePending(context.Background(), certificate, remark)
}

// finalizePendingHistoryApproved applies hours to the student (if applicable)
// and updates the pending HourChangeHistory for the given upload to approved.
// ต้องมี hour history record อยู่แล้ว ไม่งั้น error
func finalizePendingHistoryApproved(ctx context.Context, upload *models.UploadCertificate, course models.Course) error {
	// ตรวจสอบว่ามี hourHistoryId
	if upload.HourHistoryId == nil {
		return fmt.Errorf("upload certificate %s does not have hourHistoryId", upload.ID.Hex())
	}

	// determine skill type
	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	// Get student data for major-based hour limits
	student, err := students.GetStudentById(upload.StudentId)
	if err != nil {
		return fmt.Errorf("student not found: %v", err)
	}

	// คำนวณชั่วโมงที่สามารถเพิ่มได้
	currentCertHours, err := calculateCurrentCertificateHours(ctx, upload.StudentId, skillType)
	if err != nil {
		return fmt.Errorf("failed to calculate current certificate hours: %v", err)
	}

	maxTrainingHours := getMaxTrainingHours(skillType, student.Major)
	hoursToAdd := calculateHoursToAdd(course.Hour, currentCertHours, maxTrainingHours, student.Code, skillType)

	// Log hours information (ไม่อัพเดท softSkill/hardSkill โดยตรงอีกต่อไป - ใช้ hour history เป็นแหล่งข้อมูลหลัก)
	if !upload.IsDuplicate && course.IsActive {
		if hoursToAdd > 0 {
			fmt.Printf("✅ Added %d hours (%s skill) to student %s for certificate %s (max: %d, current: %d)\n",
				hoursToAdd, skillType, student.Code, upload.ID.Hex(), maxTrainingHours, currentCertHours+hoursToAdd)
		} else {
			fmt.Printf("ℹ️ No hours added to student %s (already at max %s training hours: %d/%d)\n",
				student.Code, skillType, currentCertHours, maxTrainingHours)
		}
	}

	// ใช้ hourHistoryId จาก upload certificate
	histFilter := bson.M{
		"_id":        *upload.HourHistoryId,
		"sourceType": "certificate",
		"sourceId":   upload.ID,
		"studentId":  upload.StudentId,
	}

	histUpdate := bson.M{"$set": bson.M{
		"status":     models.HCStatusApproved,
		"hourChange": hoursToAdd, // ใช้ชั่วโมงที่คำนวณแล้ว (อาจถูก cap)
		"remark":     "อนุมัติใบรับรอง",
		"changeAt":   time.Now(),
		"title":      course.Name,
		"studentId":  upload.StudentId,
		"skillType":  skillType,
	}}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if err != nil {
		return fmt.Errorf("failed to update hour history: %v", err)
	}

	// ถ้าไม่เจอ record ให้ error
	if result.MatchedCount == 0 {
		return fmt.Errorf("hour history record not found for certificate %s (hourHistoryId: %s)",
			upload.ID.Hex(), upload.HourHistoryId.Hex())
	}

	fmt.Printf("📝 Updated hour history to approved for certificate %s (ID: %s)\n",
		upload.ID.Hex(), upload.HourHistoryId.Hex())

	return nil
}

// finalizePendingHistoryRejected updates the pending HourChangeHistory to rejected.
// ต้องมี hour history record อยู่แล้ว ไม่งั้น error
func finalizePendingHistoryRejected(ctx context.Context, upload *models.UploadCertificate, course models.Course, remark string) error {
	// ตรวจสอบว่ามี hourHistoryId
	if upload.HourHistoryId == nil {
		return fmt.Errorf("upload certificate %s does not have hourHistoryId", upload.ID.Hex())
	}

	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	// ใช้ hourHistoryId จาก upload certificate
	histFilter := bson.M{
		"_id":        *upload.HourHistoryId,
		"sourceType": "certificate",
		"sourceId":   upload.ID,
		"studentId":  upload.StudentId,
	}

	histUpdate := bson.M{"$set": bson.M{
		"status":     models.HCStatusRejected,
		"hourChange": 0,
		"remark":     remark,
		"changeAt":   time.Now(),
		"title":      course.Name,
		"studentId":  upload.StudentId,
		"skillType":  skillType,
	}}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if err != nil {
		return fmt.Errorf("failed to update hour history: %v", err)
	}

	// ถ้าไม่เจอ record ให้ error
	if result.MatchedCount == 0 {
		return fmt.Errorf("hour history record not found for certificate %s (hourHistoryId: %s)",
			upload.ID.Hex(), upload.HourHistoryId.Hex())
	}

	fmt.Printf("📝 Updated hour history to rejected for certificate %s (ID: %s)\n",
		upload.ID.Hex(), upload.HourHistoryId.Hex())

	return nil
}

// updateCertificateHoursApproved applies approval logic (wraps finalizePendingHistoryApproved)
// This function is used by admin flow to add hours when a certificate is approved.
func updateCertificateHoursApproved(ctx context.Context, certificate *models.UploadCertificate) error {
	// Load course
	course, err := courses.GetCourseByID(certificate.CourseId)
	if err != nil {
		return fmt.Errorf("course not found: %v", err)
	}

	// Finalize the pending history to approved (this will compute hourChange and update HourChangeHistory)
	if err := finalizePendingHistoryApproved(ctx, certificate, *course); err != nil {
		return fmt.Errorf("failed to finalize pending history approved: %v", err)
	}

	// After updating hour history, update student aggregated status/hours
	if err := hourhistory.UpdateStudentStatus(ctx, certificate.StudentId); err != nil {
		fmt.Printf("⚠️ Warning: Failed to update student status after approving certificate %s: %v\n", certificate.ID.Hex(), err)
		// don't fail the whole operation because hours were applied
	}

	return nil
}

// CheckStudentCourse resolves student and course by hex IDs and returns them.
func CheckStudentCourse(studentHex string, courseHex string) (*models.Student, *models.Course, error) {
	sid, err := primitive.ObjectIDFromHex(studentHex)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid student id: %v", err)
	}
	cid, err := primitive.ObjectIDFromHex(courseHex)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid course id: %v", err)
	}

	student, err := students.GetStudentById(sid)
	if err != nil {
		return nil, nil, fmt.Errorf("student not found: %v", err)
	}

	course, err := courses.GetCourseByID(cid)
	if err != nil {
		return nil, nil, fmt.Errorf("course not found: %v", err)
	}

	return student, course, nil
}

// checkDuplicateURL checks if an approved upload certificate already exists for the given URL.
// If currentID is non-nil, that ID will be excluded from the search (useful when re-checking the same record).
func checkDuplicateURL(url string, studentID primitive.ObjectID, courseID primitive.ObjectID, currentID *primitive.ObjectID) (bool, *models.UploadCertificate, error) {
	ctx := context.Background()
	filter := bson.M{"url": url, "status": models.StatusApproved}
	if currentID != nil {
		filter["_id"] = bson.M{"$ne": *currentID}
	}

	var existing models.UploadCertificate
	err := DB.UploadCertificateCollection.FindOne(ctx, filter).Decode(&existing)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &existing, nil
}

// BuuMooc fetches the page HTML and calls the BUUMooc fastapi endpoint for verification.
func BuuMooc(url string, student *models.Student, course *models.Course) (*FastAPIResp, error) {
	fastapi := os.Getenv("FASTAPI_URL")
	if fastapi == "" {
		return nil, fmt.Errorf("FASTAPI_URL not configured")
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return callBUUMoocFastAPI(fastapi, string(body), student.Name, student.EngName, course.CertificateName, course.CertificateNameEN)
}

// ThaiMooc fetches the resource (pdf/html) and calls the ThaiMooc fastapi endpoint.
func ThaiMooc(url string, student *models.Student, course *models.Course) (*FastAPIResp, error) {
	fastapi := os.Getenv("FASTAPI_URL")
	if fastapi == "" {
		return nil, fmt.Errorf("FASTAPI_URL not configured")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return callThaiMoocFastAPI(fastapi, data, student.Name, student.EngName, course.CertificateName, course.CertificateNameEN)
}
