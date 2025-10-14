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
	SubmissionID     *primitive.ObjectID `json:"submissionId,omitempty" bson:"submissionId,omitempty"`
	AttendedAllDays  *bool               `json:"attendedAllDays,omitempty" bson:"attendedAllDays,omitempty"`
}

// SuccessResponse ใช้เป็นโครงสร้าง JSON Response ที่ Swagger ใช้
type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type BulkEnrollItem struct {
	StudentCode string  `json:"studentCode"`
	Food        *string `json:"food"`
}

type BulkEnrollRequest struct {
	ProgramItemID string           `json:"programItemId"`
	Students      []BulkEnrollItem `json:"students"`
}

type BulkEnrollResult struct {
	ProgramItemID  string                  `json:"programItemId"`
	TotalRequested int                     `json:"totalRequested"`
	Success        []BulkEnrollSuccessItem `json:"success"`
	Failed         []BulkEnrollFailedItem  `json:"failed"`
}

type BulkEnrollSuccessItem struct {
	StudentCode string `json:"studentCode"`
	StudentID   string `json:"studentId"`
	Message     string `json:"message"`
}
type BulkEnrollFailedItem struct {
	StudentCode string `json:"studentCode"`
	Reason      string `json:"reason"`
}
