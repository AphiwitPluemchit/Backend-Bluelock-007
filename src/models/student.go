package models

import (
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
	Year      string             `bson:"year" json:"year"` // ปีการศึกษา เช่น "2567"
}
