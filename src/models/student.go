package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Student นิสิต
type Student struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Code      string             `json:"code" bson:"code"`
	Name      string             `json:"name" bson:"name"`
	Email     string             `json:"email,omitempty" bson:"email,omitempty"`
	Status    int                `json:"status" bson:"status"`        // 0 = พ้นสภาพ, 1 = ชั่วโมงน้อยมาก, 2 = ชั่วโมงน้อย, 3 = ชั่วโมงครบแล้ว
	Password  string             `json:"-" bson:"password,omitempty"` // ไม่ให้ส่ง Password ออกไป
	SoftSkill int                `json:"softSkill,omitempty" bson:"softSkill,omitempty"`
	HardSkill int                `json:"hardSkill,omitempty" bson:"hardSkill,omitempty"`
	Major     string             `json:"major,omitempty" bson:"major,omitempty"`
}
