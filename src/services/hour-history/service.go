package hourhistory

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SaveHourHistory บันทึกประวัติการเปลี่ยนแปลงชั่วโมง
func SaveHourHistory(
	ctx context.Context,
	studentID primitive.ObjectID,
	skillType string, // "soft" | "hard"
	hourChange int, // บวก = เพิ่ม, ลบ = ลด
	title string,
	remark string,
	sourceType string, // "program" | "certificate"
	sourceID primitive.ObjectID,
	enrollmentID *primitive.ObjectID, // optional, สำหรับ program เท่านั้น
) error {
	history := models.HourChangeHistory{
		ID:           primitive.NewObjectID(),
		SkillType:    skillType,
		HourChange:   hourChange,
		Remark:       remark,
		ChangeAt:     time.Now(),
		Title:        title,
		StudentID:    studentID,
		EnrollmentID: enrollmentID,
		SourceType:   sourceType,
		SourceID:     sourceID,
	}

	if _, err := DB.HourChangeHistoryCollection.InsertOne(ctx, history); err != nil {
		return fmt.Errorf("failed to save hour change history: %v", err)
	}

	return nil
}

// GetHistoryByStudent ดึงประวัติการเปลี่ยนแปลงชั่วโมงของนิสิต
func GetHistoryByStudent(ctx context.Context, studentID primitive.ObjectID) ([]models.HourChangeHistory, error) {
	cursor, err := DB.HourChangeHistoryCollection.Find(ctx, primitive.M{"studentId": studentID})
	if err != nil {
		return nil, fmt.Errorf("failed to get hour history: %v", err)
	}
	defer cursor.Close(ctx)

	var histories []models.HourChangeHistory
	if err := cursor.All(ctx, &histories); err != nil {
		return nil, fmt.Errorf("failed to decode hour history: %v", err)
	}

	return histories, nil
}

// GetHistoryBySource ดึงประวัติการเปลี่ยนแปลงชั่วโมงตาม source (program/certificate)
func GetHistoryBySource(ctx context.Context, sourceType string, sourceID primitive.ObjectID) ([]models.HourChangeHistory, error) {
	cursor, err := DB.HourChangeHistoryCollection.Find(ctx, primitive.M{
		"sourceType": sourceType,
		"sourceId":   sourceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get hour history: %v", err)
	}
	defer cursor.Close(ctx)

	var histories []models.HourChangeHistory
	if err := cursor.All(ctx, &histories); err != nil {
		return nil, fmt.Errorf("failed to decode hour history: %v", err)
	}

	return histories, nil
}
