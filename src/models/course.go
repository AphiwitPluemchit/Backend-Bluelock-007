package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Course struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty" swaggertype:"string" example:"507f1f77bcf86cd799439011"`
	Name        string             `json:"name" bson:"name" example:"Introduction to Programming"`
	Description string             `json:"description" bson:"description" example:"Learn the basics of programming with this introductory course"`
	Date        time.Time          `json:"date" bson:"date" example:"2025-07-19T00:00:00Z"`
	Issuer      string             `json:"issuer" bson:"issuer" example:"Computer Science Department"`
	Type        string             `json:"type" bson:"type" example:"lms" enums:"lms,buumooc,thaimooc"`
	Hour        int                `json:"hour" bson:"hour" example:"4"`
	IsHardSkill bool               `json:"isHardSkill" bson:"isHardSkill" example:"true"` // true = hard skill, false = soft skill
	IsActive    bool               `json:"isActive" bson:"isActive" example:"true"`
}
