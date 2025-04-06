package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CheckInOut การเช็คชื่อ
type CheckInOut struct {
	ID             primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	CheckIn        *time.Time         `json:"checkIn" bson:"checkIn"`
	CheckOut       *time.Time         `json:"checkOut" bson:"checkOut"`
	Status         *string            `json:"status" bson:"status"`
	ActivityItemID primitive.ObjectID `json:"activityItemId" bson:"activityItemId"`
	StudentID      primitive.ObjectID `json:"studentId" bson:"studentId"`
}
