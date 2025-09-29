package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
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

// helper: max int
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
