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

// HourChangeHistory บันทึกประวัติการเปลี่ยนแปลงชั่วโมง
type HourChangeHistory struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	SkillType    string              `bson:"skillType" json:"skillType"`                           // "soft" | "hard"
	HourChange   int                 `bson:"hourChange" json:"hourChange"`                         // จำนวนชั่วโมงที่เปลี่ยน (บวก = เพิ่ม, ลบ = ลด)
	Remark       string              `bson:"remark,omitempty" json:"remark,omitempty"`             // หมายเหตุ
	ChangeAt     time.Time           `bson:"changeAt" json:"changeAt"`                             // เวลาที่เกิดการเปลี่ยนแปลง
	Title        string              `bson:"title" json:"title"`                                   // หัวข้อ/ชื่อของการเปลี่ยนแปลง
	StudentID    primitive.ObjectID  `bson:"studentId" json:"studentId"`                           // นิสิตที่ได้รับผลกระทบ
	EnrollmentID *primitive.ObjectID `bson:"enrollmentId,omitempty" json:"enrollmentId,omitempty"` // enrollment ID (สำหรับ program)
	SourceType   string              `bson:"sourceType" json:"sourceType"`                         // "program" | "certificate"
	SourceID     primitive.ObjectID  `bson:"sourceId" json:"sourceId"`                             // ID ของ program/certificate ที่เป็นต้นเหตุ
}

// CheckinoutRecord สำหรับการแสดงข้อมูลการเช็คชื่อ
type CheckinoutRecord struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Checkin       *time.Time         `bson:"checkin" json:"checkin"`
	Checkout      *time.Time         `bson:"checkout" json:"checkout"`
	Participation *string            `bson:"participation" json:"participation" example:"ยังไม่เข้าร่วมกิจกรรม, เช็คอิน/เช็คเอาท์ตรงเวลา, เช็คอิน/เช็คเอาท์ไม่ตรงเวลา, เช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์"`
}
