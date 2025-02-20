package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CheckInOut การเช็คชื่อ
type CheckInOut struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	CheckIn        time.Time          `bson:"checkIn"`
	CheckOut       time.Time          `bson:"checkOut"`
	Status         string             `bson:"status"`
	ActivityItemID primitive.ObjectID `bson:"activityItemId"`
	EnrollmentID   primitive.ObjectID `bson:"enrollmentId"`
}
