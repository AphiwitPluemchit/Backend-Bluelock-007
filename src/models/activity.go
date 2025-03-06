package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Activity กิจกรรมหลัก
type Activity struct {
	ID            primitive.ObjectID   `json:"id,omitempty" bson:"_id,omitempty"`
	Name          *string              `json:"name" bson:"name" validate:"required" example:"Football Tournament"`
	Type          string               `json:"type" bson:"type" validate:"required" example:"one"`
	ActivityState string               `json:"activityState" bson:"activityState" validate:"required" example:"planning"`
	Skill         string               `json:"skill" bson:"skill" validate:"required" example:"hard"`
	File          string               `json:"file" bson:"file" validate:"required"  example:"image.jpg"`
	MajorIDs      []primitive.ObjectID `json:"majorIds" bson:"majorIds" validate:"required" example:"67bf0bd48873e448798fed34,67bf0bda8873e448798fed35"`
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
	ActivityState string             `json:"activityState" bson:"activityState" validate:"required"  example:"planning"`
	Skill         string             `json:"skill" bson:"skill" validate:"required" example:"hard"`
	File          string             `json:"file" bson:"file" validate:"required"  example:"image.jpg"`
	Majors        []Major            `json:"majors" bson:"majors" validate:"required"`
	ActivityItems []ActivityItem     `json:"activityItems"`
}
