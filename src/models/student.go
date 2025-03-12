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
	Status    string             `json:"status" bson:"status"`
	Password  string             `json:"-" bson:"password,omitempty"` // ไม่ให้ส่ง Password ออกไป
	SoftSkill int                `json:"softSkill,omitempty" bson:"softSkill,omitempty"`
	HardSkill int                `json:"hardSkill,omitempty" bson:"hardSkill,omitempty"`
	MajorID   primitive.ObjectID `json:"majorId,omitempty" bson:"majorId,omitempty"`
	Major     Major              `json:"major,omitempty" `
}
