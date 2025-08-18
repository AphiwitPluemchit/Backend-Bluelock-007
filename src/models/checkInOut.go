package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CheckInOut การเช็คชื่อ
// เปลี่ยน ActivityItemID -> ActivityID
type CheckInOut struct {
	ID         primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	CheckIn    *time.Time         `json:"checkIn" bson:"checkIn"`
	CheckOut   *time.Time         `json:"checkOut" bson:"checkOut"`
	Status     *string            `json:"status" bson:"status"`
	ActivityID primitive.ObjectID `json:"activityId" bson:"activityId"`
	StudentID  primitive.ObjectID `json:"studentId" bson:"studentId"`
}

// QRToken สำหรับเก็บ token ของ QR code
// { token, activityId, createdAt, expiresAt, claimedByStudentId (nullable) }
type QRToken struct {
	Token              string              `bson:"token" json:"token"`
	ActivityID         primitive.ObjectID  `bson:"activityId" json:"activityId"`
	Type               string              `bson:"type" json:"type"`
	CreatedAt          int64               `bson:"createdAt" json:"createdAt"`
	ExpiresAt          int64               `bson:"expiresAt" json:"expiresAt"`
	ClaimedByStudentID *primitive.ObjectID `bson:"claimedByStudentId,omitempty" json:"claimedByStudentId,omitempty"`
}

// CheckinRecord สำหรับเก็บข้อมูลการเช็คชื่อ
// { studentId, activityId, type: 'checkin' | 'checkout', timestamp }
type CheckinRecord struct {
	StudentID  primitive.ObjectID `bson:"studentId" json:"studentId"`
	ActivityID primitive.ObjectID `bson:"activityId" json:"activityId"`
	Type       string             `bson:"type" json:"type"`
	Timestamp  int64              `bson:"timestamp" json:"timestamp"`
}

// QRClaim สำหรับเก็บข้อมูลการ claim QR ใน MongoDB
// { token, studentId, activityId, type, claimedAt, expireAt }
type QRClaim struct {
	Token      string             `bson:"token" json:"token"`
	StudentID  primitive.ObjectID `bson:"studentId" json:"studentId"`
	ActivityID primitive.ObjectID `bson:"activityId" json:"activityId"`
	Type       string             `bson:"type" json:"type"`
	ClaimedAt  time.Time          `bson:"claimedAt" json:"claimedAt"`
	ExpireAt   time.Time          `bson:"expireAt" json:"expireAt"`
}
type CheckinoutRecord struct {
	Checkin  *time.Time `bson:"checkin" json:"checkin"`
	Checkout *time.Time `bson:"checkout" json:"checkout"`
}
