package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Student นิสิต
type Student struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code      string             `bson:"code" json:"code"`
	Name      string             `bson:"name" json:"name"`
	EngName   string             `bson:"engName" json:"engName"`
	Status    int                `bson:"status" json:"status"` // 0พ้นสภาพ 1ชั่วโมงน้อยมาก 2ชั่วโมงน้อย 3ชั่วโมงครบแล้ว 4ออกผึกแล้ว
	SoftSkill int                `bson:"softSkill" json:"softSkill"`
	HardSkill int                `bson:"hardSkill" json:"hardSkill"`
	Major     string             `bson:"major" json:"major"`
}

// HourChangeHistory ประวัติการเปลี่ยนแปลงชั่วโมงของนักเรียน
type HourChangeHistory struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentID       primitive.ObjectID `bson:"studentId" json:"studentId"`
	StudentName     string             `bson:"studentName" json:"studentName"`
	StudentCode     string             `bson:"studentCode" json:"studentCode"`
	ProgramID       primitive.ObjectID `bson:"programId" json:"programId"`
	ProgramName     string             `bson:"programName" json:"programName"`
	ProgramItemID   primitive.ObjectID `bson:"programItemId" json:"programItemId"`
	ProgramItemName string             `bson:"programItemName" json:"programItemName"`
	SkillType       string             `bson:"skillType" json:"skillType"`     // "soft" หรือ "hard"
	HoursChange     int                `bson:"hoursChange" json:"hoursChange"` // บวก = เพิ่ม, ลบ = ลด
	ChangeType      string             `bson:"changeType" json:"changeType"`   // "add", "remove", "no_change"
	Reason          string             `bson:"reason" json:"reason"`           // เหตุผลการเปลี่ยนแปลง
	ChangedAt       time.Time          `bson:"changedAt" json:"changedAt"`
}
