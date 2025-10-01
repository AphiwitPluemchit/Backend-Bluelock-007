package summary_reports

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ctx = context.Background()

// CreateSummaryReport สร้าง summary report สำหรับ program ใหม่
func CreateSummaryReport(programID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ตรวจสอบว่า program มีอยู่จริงหรือไม่
	var program models.Program
	err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programID}).Decode(&program)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("program not found")
		}
		return err
	}

	// ดึงข้อมูล programItems ทั้งหมด
	cursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID})
	if err != nil {
		return fmt.Errorf("failed to find program items: %w", err)
	}
	defer cursor.Close(ctx)

	var programItems []models.ProgramItem
	if err = cursor.All(ctx, &programItems); err != nil {
		return fmt.Errorf("failed to decode program items: %w", err)
	}

	// สร้าง summary report สำหรับแต่ละ programItem และแต่ละ date
	var summariesToInsert []interface{}
	for _, item := range programItems {
		for _, date := range item.Dates {
			summary := models.Summary_Check_In_Out_Reports{
				ID:               primitive.NewObjectID(),
				ProgramID:        programID,
				Date:             date.Date,
				Registered:       0,
				Checkin:          0,
				CheckinLate:      0,
				Checkout:         0,
				NotParticipating: 0,
			}
			summariesToInsert = append(summariesToInsert, summary)
		}
	}

	// บันทึกลงฐานข้อมูลทั้งหมดในครั้งเดียว
	if len(summariesToInsert) > 0 {
		_, err = DB.SummaryCheckInOutReportsCollection.InsertMany(ctx, summariesToInsert)
		if err != nil {
			return fmt.Errorf("failed to create summary reports: %w", err)
		}
	}

	log.Printf("✅ Created %d summary reports for program: %s", len(summariesToInsert), programID.Hex())
	return nil
}

// UpdateRegisteredCount อัปเดตจำนวนผู้ลงทะเบียนสำหรับ programItem และ date ที่ระบุ
func UpdateRegisteredCount(programItemID primitive.ObjectID, date string, change int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ดึงข้อมูล programItem เพื่อหา programID
	var programItem models.ProgramItem
	err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": programItemID}).Decode(&programItem)
	if err != nil {
		return fmt.Errorf("failed to find program item: %w", err)
	}

	// อัปเดต registered count สำหรับ programItem และ date ที่ระบุ
	filter := bson.M{
		"programId": programItem.ProgramID,
		"date":      date,
	}
	update := bson.M{
		"$inc": bson.M{"registered": change},
	}

	result, err := DB.SummaryCheckInOutReportsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update registered count: %w", err)
	}

	if result.ModifiedCount == 0 {
		return errors.New("summary report not found for this program item and date")
	}

	// อัปเดต NotParticipating count ด้วย
	// NotParticipating = Registered - (Checkin + CheckinLate)
	err = RecalculateNotParticipating(programItem.ProgramID, date)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to recalculate NotParticipating: %v", err)
	}

	log.Printf("✅ Updated registered count for program %s, item %s, date %s by %d",
		programItem.ProgramID.Hex(), programItemID.Hex(), date, change)
	return nil
}

// UpdateCheckinCount อัปเดตจำนวนการเช็คอิน (ตรงเวลาหรือสาย) สำหรับ date ที่ระบุ
func UpdateCheckinCount(programID primitive.ObjectID, date string, isLate bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"programId": programID,
		"date":      date,
	}

	var update bson.M
	if isLate {
		update = bson.M{"$inc": bson.M{"checkinLate": 1}}
	} else {
		update = bson.M{"$inc": bson.M{"checkin": 1}}
	}

	result, err := DB.SummaryCheckInOutReportsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update checkin count: %w", err)
	}

	if result.ModifiedCount == 0 {
		return errors.New("summary report not found for this program and date")
	}

	// ลด NotParticipating count
	err = RecalculateNotParticipating(programID, date)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to recalculate NotParticipating: %v", err)
	}

	checkinType := "on-time"
	if isLate {
		checkinType = "late"
	}
	log.Printf("✅ Updated %s checkin count for program %s, date %s", checkinType, programID.Hex(), date)
	return nil
}

// UpdateCheckoutCount อัปเดตจำนวนการเช็คเอาท์ สำหรับ date ที่ระบุ
func UpdateCheckoutCount(programID primitive.ObjectID, date string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"programId": programID,
		"date":      date,
	}
	update := bson.M{"$inc": bson.M{"checkout": 1}}

	result, err := DB.SummaryCheckInOutReportsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update checkout count: %w", err)
	}

	if result.ModifiedCount == 0 {
		return errors.New("summary report not found for this program and date")
	}

	log.Printf("✅ Updated checkout count for program %s, date %s", programID.Hex(), date)
	return nil
}

// RecalculateNotParticipating คำนวณ NotParticipating ใหม่สำหรับ date ที่ระบุ
// NotParticipating = Registered - (Checkin + CheckinLate)
func RecalculateNotParticipating(programID primitive.ObjectID, date string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ดึงข้อมูลปัจจุบัน
	var summary models.Summary_Check_In_Out_Reports
	filter := bson.M{
		"programId": programID,
		"date":      date,
	}
	err := DB.SummaryCheckInOutReportsCollection.FindOne(ctx, filter).Decode(&summary)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("summary report not found for this program and date")
		}
		return err
	}

	// คำนวณ NotParticipating ใหม่
	notParticipating := summary.Registered - (summary.Checkin + summary.CheckinLate)

	// ตรวจสอบให้แน่ใจว่าไม่เป็นค่าลบ
	if notParticipating < 0 {
		notParticipating = 0
	}

	// อัปเดต
	update := bson.M{"$set": bson.M{"notParticipating": notParticipating}}
	_, err = DB.SummaryCheckInOutReportsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update notParticipating count: %w", err)
	}

	log.Printf("✅ Recalculated NotParticipating for program %s, date %s: %d", programID.Hex(), date, notParticipating)
	return nil
}

// GetSummaryReport ดึงข้อมูล summary report ของ program
func GetSummaryReport(programID primitive.ObjectID) ([]models.Summary_Check_In_Out_Reports, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := DB.SummaryCheckInOutReportsCollection.Find(ctx, bson.M{"programId": programID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch summary reports: %w", err)
	}
	defer cursor.Close(ctx)

	var summaries []models.Summary_Check_In_Out_Reports
	if err = cursor.All(ctx, &summaries); err != nil {
		return nil, fmt.Errorf("failed to decode summary reports: %w", err)
	}

	return summaries, nil
}

// GetSummaryReportByDate ดึงข้อมูล summary report ของ program และ date ที่ระบุ
func GetSummaryReportByDate(programID primitive.ObjectID, date string) (*models.Summary_Check_In_Out_Reports, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var summary models.Summary_Check_In_Out_Reports
	filter := bson.M{
		"programId": programID,
		"date":      date,
	}

	err := DB.SummaryCheckInOutReportsCollection.FindOne(ctx, filter).Decode(&summary)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("summary report not found for this program and date")
		}
		return nil, err
	}

	return &summary, nil
}

// GetAllSummaryReports ดึงข้อมูล summary reports ทั้งหมด
func GetAllSummaryReports() ([]models.Summary_Check_In_Out_Reports, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := DB.SummaryCheckInOutReportsCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch summary reports: %w", err)
	}
	defer cursor.Close(ctx)

	var summaries []models.Summary_Check_In_Out_Reports
	if err = cursor.All(ctx, &summaries); err != nil {
		return nil, fmt.Errorf("failed to decode summary reports: %w", err)
	}

	return summaries, nil
}

// DeleteSummaryReport ลบ summary report ของ program
func DeleteSummaryReport(programID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"programId": programID}
	result, err := DB.SummaryCheckInOutReportsCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete summary report: %w", err)
	}

	if result.DeletedCount == 0 {
		return errors.New("summary report not found for this program")
	}

	log.Printf("✅ Deleted summary report for program %s", programID.Hex())
	return nil
}

// EnsureSummaryReportExists ตรวจสอบและสร้าง summary report ถ้ายังไม่มี
func EnsureSummaryReportExists(programID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ตรวจสอบว่ามี summary report อยู่แล้วหรือไม่
	count, err := DB.SummaryCheckInOutReportsCollection.CountDocuments(ctx, bson.M{"programId": programID})
	if err != nil {
		return fmt.Errorf("failed to check existing summary report: %w", err)
	}

	// ถ้าไม่มี ให้สร้างใหม่
	if count == 0 {
		return CreateSummaryReport(programID)
	}

	return nil
}

// EnsureSummaryReportExistsForDate ตรวจสอบและสร้าง summary report สำหรับ date ที่ระบุถ้ายังไม่มี
func EnsureSummaryReportExistsForDate(programID primitive.ObjectID, date string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ตรวจสอบว่ามี summary report สำหรับ date นี้อยู่แล้วหรือไม่
	count, err := DB.SummaryCheckInOutReportsCollection.CountDocuments(ctx, bson.M{
		"programId": programID,
		"date":      date,
	})
	if err != nil {
		return fmt.Errorf("failed to check existing summary report for date: %w", err)
	}

	// ถ้าไม่มี ให้สร้างใหม่สำหรับ date นี้
	if count == 0 {
		summary := models.Summary_Check_In_Out_Reports{
			ID:               primitive.NewObjectID(),
			ProgramID:        programID,
			Date:             date,
			Registered:       0,
			Checkin:          0,
			CheckinLate:      0,
			Checkout:         0,
			NotParticipating: 0,
		}

		_, err = DB.SummaryCheckInOutReportsCollection.InsertOne(ctx, summary)
		if err != nil {
			return fmt.Errorf("failed to create summary report for date: %w", err)
		}

		log.Printf("✅ Created summary report for program %s, date %s", programID.Hex(), date)
	}

	return nil
}

// DeleteAllSummaryReportsForProgram ลบ summary reports ทั้งหมดของ program
func DeleteAllSummaryReportsForProgram(programID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"programId": programID}
	result, err := DB.SummaryCheckInOutReportsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete summary reports for program: %w", err)
	}

	log.Printf("✅ Deleted %d summary reports for program %s", result.DeletedCount, programID.Hex())
	return nil
}
