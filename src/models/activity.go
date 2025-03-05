package models

import (
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ObjectIDNullable รองรับ `null` ถ้า ObjectID เป็นค่าที่ว่าง
type ObjectIDNullable primitive.ObjectID

// MarshalJSON ปรับ ObjectID ให้เป็น `null` ถ้าเป็น `000000000000000000000000`
func (o ObjectIDNullable) MarshalJSON() ([]byte, error) {
	oid := primitive.ObjectID(o)
	if oid.IsZero() {
		return json.Marshal(nil) // ส่ง `null` แทน
	}
	return json.Marshal(oid.Hex()) // ส่งเป็น String ปกติ
}

// Activity กิจกรรมหลัก
type Activity struct {
	ID              primitive.ObjectID   `json:"id,omitempty" bson:"_id,omitempty"`
	Name            *string              `json:"name" bson:"name" validate:"required" example:"Football Tournament"`
	Type            string               `json:"type" bson:"type" validate:"required" example:"one"`
	ActivityStateID primitive.ObjectID   `json:"activityStateId" bson:"activityStateId" validate:"required" example:"67bf1cdd95fb769b3ded079e"`
	SkillID         primitive.ObjectID   `json:"skillId" bson:"skillId" validate:"required" example:"67bf18532b62df84b60d95a2"`
	MajorIDs        []primitive.ObjectID `json:"majorIds" bson:"majorIds" validate:"required" example:"67bf0bd48873e448798fed34,67bf0bda8873e448798fed35"`
}

// ActivityItem รายละเอียดกิจกรรมย่อย
type ActivityItem struct {
	ID              primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ActivityID      primitive.ObjectID `json:"activityId,omitempty" bson:"activityId,omitempty"`
	Name            *string            `json:"name" bson:"name" validate:"required" example:"Quarter Final"`
	MaxParticipants *int               `json:"maxParticipants" bson:"maxParticipants" validate:"required,min=1" example:"22"`
	Room            *string            `json:"room" bson:"room" validate:"required" example:"Stadium A"`
	StartDate       *string            `json:"startDate" bson:"startDate" validate:"required" example:"2025-03-10"`
	EndDate         *string            `json:"endDate" bson:"endDate" validate:"required" example:"2025-03-11"`
	Duration        *int               `json:"duration" bson:"duration" validate:"required,min=1" example:"2"`
	Hour            *int               `json:"hour" bson:"hour" validate:"required,min=1" example:"4"`
}

// validate ยังไม่ถูกใช้งาน เพราะยังไม่อยากให้มันยังไม่ติด validate

// RequestCreateActivity ใช้สำหรับ CreateActivity API
type ActivityDto struct {
	ID            primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name          *string            `json:"name" bson:"name" validate:"required" example:"Football Tournament"`
	Type          string             `json:"type" bson:"type" validate:"required" example:"one"`
	ActivityState ActivityState      `json:"activityState" bson:"activityState" validate:"required"`
	Skill         Skill              `json:"skill" bson:"skill" validate:"required"`
	Majors        []Major            `json:"majors" bson:"majors" validate:"required"`
	ActivityItems []ActivityItem     `json:"activityItems"`
}
