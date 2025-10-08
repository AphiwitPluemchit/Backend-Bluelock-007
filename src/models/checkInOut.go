package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CheckInOut การเช็คชื่อ
// เปลี่ยน ProgramItemID -> ProgramID
type CheckInOut struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	CheckIn   *time.Time         `json:"checkIn" bson:"checkIn"`
	CheckOut  *time.Time         `json:"checkOut" bson:"checkOut"`
	Status    *string            `json:"status" bson:"status"`
	ProgramID primitive.ObjectID `json:"programId" bson:"programId"`
	StudentID primitive.ObjectID `json:"studentId" bson:"studentId"`
}

// QRToken สำหรับเก็บ token ของ QR code
// { token, programId, createdAt, expiresAt, claimedByStudentId (nullable) }
type QRToken struct {
	Token              string              `bson:"token" json:"token"`
	ProgramID          primitive.ObjectID  `bson:"programId" json:"programId"`
	Type               string              `bson:"type" json:"type"`
	CreatedAt          int64               `bson:"createdAt" json:"createdAt"`
	ExpiresAt          int64               `bson:"expiresAt" json:"expiresAt"`
	ClaimedByStudentID *primitive.ObjectID `bson:"claimedByStudentId,omitempty" json:"claimedByStudentId,omitempty"`
}

// CheckinRecord สำหรับเก็บข้อมูลการเช็คชื่อ
// { studentId, programId, type: 'checkin' | 'checkout', timestamp }
type CheckinRecord struct {
	StudentID     primitive.ObjectID `bson:"studentId" json:"studentId"`
	ProgramItemID primitive.ObjectID `json:"programItemId" bson:"programItemId"`
	Type          string             `bson:"type" json:"type"`
	Timestamp     time.Time          `bson:"timestamp" json:"timestamp"`
}

// QRClaim สำหรับเก็บข้อมูลการ claim QR ใน MongoDB
// { token, studentId, programId, type, claimedAt, expireAt }
type QRClaim struct {
	Token     string             `bson:"token" json:"token"`
	StudentID primitive.ObjectID `bson:"studentId" json:"studentId"`
	ProgramID primitive.ObjectID `bson:"programId" json:"programId"`
	Type      string             `bson:"type" json:"type"`
	ClaimedAt time.Time          `bson:"claimedAt" json:"claimedAt"`
	ExpireAt  time.Time          `bson:"expireAt" json:"expireAt"`
}

// models/hour_change_history.go
type HourChangeHistory struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentID   primitive.ObjectID `bson:"studentId" json:"studentId"`
	StudentCode string             `bson:"studentCode,omitempty" json:"studentCode,omitempty"`

	// สำหรับกิจกรรม (program)
	ProgramID     primitive.ObjectID  `bson:"programId,omitempty" json:"programId,omitempty"`
	ProgramItemID primitive.ObjectID  `bson:"programItemId,omitempty" json:"programItemId,omitempty"`
	EnrollmentID  *primitive.ObjectID `bson:"enrollmentId,omitempty" json:"enrollmentId,omitempty"`
	ProgramDate   string              `bson:"programDate,omitempty" json:"programDate,omitempty"`

	// สำหรับ certificate
	CertificateID *primitive.ObjectID `bson:"certificateId,omitempty" json:"certificateId,omitempty"`

	Type        string    `bson:"type" json:"type"` // "program" | "certificate"
	SkillType   string    `bson:"skillType" json:"skillType"`
	HoursChange int       `bson:"hoursChange" json:"hoursChange"`
	ChangeType  string    `bson:"changeType" json:"changeType"` // "add" | "remove" | "no_change"
	Remark      string    `bson:"remark,omitempty" json:"remark,omitempty"`
	ChangedAt   time.Time `bson:"changedAt" json:"changedAt"`
}

// CheckinoutRecord สำหรับการแสดงข้อมูลการเช็คชื่อ
type CheckinoutRecord struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Checkin       *time.Time         `bson:"checkin" json:"checkin"`
	Checkout      *time.Time         `bson:"checkout" json:"checkout"`
	Participation *string            `bson:"participation" json:"participation" example:"ยังไม่เข้าร่วมกิจกรรม, เช็คอิน/เช็คเอาท์ตรงเวลา, เช็คอิน/เช็คเอาท์ไม่ตรงเวลา, เช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์"`
}
