package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Enrollment การลงทะเบียน
type Enrollment struct {
	ID               primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	RegistrationDate time.Time           `json:"registrationDate" bson:"registrationDate"`
	FoodVoteID       *primitive.ObjectID `json:"foodVoteId" bson:"foodVoteId"`
	ActivityItemID   primitive.ObjectID  `json:"activityItemId" bson:"activityItemId"`
	StudentID        primitive.ObjectID  `json:"studentId" bson:"StudentId"`
}
