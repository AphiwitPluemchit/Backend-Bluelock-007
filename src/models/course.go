package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Course struct {
	ID           primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty" swaggertype:"string" example:"507f1f77bcf86cd799439011"`
	Name         string             `json:"name" bson:"name" example:"Introduction to Programming"`
	Description  string             `json:"description" bson:"description" example:"Learn the basics of programming with this introductory course"`
	Link         string             `json:"link" bson:"link" example:"https://www.example.com/course"`
	Issuer       string             `json:"issuer" bson:"issuer" example:"Computer Science Department"`
	Type         string             `json:"type" bson:"type" example:"lms" enums:"lms,buumooc,thaimooc"`
	Hour         int                `json:"hour" bson:"hour" example:"4"`
	IsHardSkill  bool               `json:"isHardSkill" bson:"isHardSkill" example:"true"` // true = hard skill, false = soft skill
	IsActive     bool               `json:"isActive" bson:"isActive" example:"true"`
}

// CourseFilters ใช้เก็บค่าการกรองสำหรับคอร์ส
type CourseFilters struct {
	Type        string `json:"type" query:"type"` // lms, buumooc, thaimooc
	IsHardSkill *bool  `json:"isHardSkill" query:"isHardSkill"`
	IsActive    *bool  `json:"isActive" query:"isActive"`
}

// CoursePaginatedResponse is a concrete type for paginated course responses
type CoursePaginatedResponse struct {
	Data []Course       `json:"data"`
	Meta PaginationMeta `json:"meta"`
}
