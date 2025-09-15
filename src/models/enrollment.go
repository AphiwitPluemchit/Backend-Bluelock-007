package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Enrollment - การลงทะเบียนกิจกรรม
type Enrollment struct {
	ID               primitive.ObjectID  `json:"id,omitempty" bson:"_id,omitempty"`
	RegistrationDate time.Time           `json:"registrationDate" bson:"registrationDate"`
	ProgramID        primitive.ObjectID  `json:"programId" bson:"programId"`
	ProgramItemID    primitive.ObjectID  `json:"programItemId" bson:"programItemId"`
	StudentID        primitive.ObjectID  `json:"studentId" bson:"studentId"`
	Food             *string             `json:"food" bson:"food"`
	CheckinoutRecord *[]CheckinoutRecord `json:"checkinoutRecord" bson:"checkinoutRecord"`
}

// SuccessResponse ใช้เป็นโครงสร้าง JSON Response ที่ Swagger ใช้
type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}
