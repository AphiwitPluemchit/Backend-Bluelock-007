package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Activity กิจกรรมหลัก
type Activity struct {
	ID            primitive.ObjectID   `json:"id,omitempty" bson:"_id,omitempty"`
	Name          *string              `json:"name" bson:"name" example:"Football Tournament"`
	Type          string               `json:"type" bson:"type" example:"one"`
	ActivityState string               `json:"activityState" bson:"activityState" example:"planning"`
	Skill         string               `json:"skill" bson:"skill" example:"hard"`
	File          string               `json:"file" bson:"file"  example:"image.jpg"`
	StudentYears  []int                `json:"studentYears" bson:"studentYears" example:"1,2,3,4"`
	MajorIDs      []primitive.ObjectID `json:"majorIds" bson:"majorIds" example:"67bf0bd48873e448798fed34,67bf0bda8873e448798fed35"`
	FoodVotes     []FoodVote           `json:"foodVotes"`
}

// ActivityItem รายละเอียดกิจกรรมย่อย
type ActivityItem struct {
	ID              primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ActivityID      primitive.ObjectID `json:"activityId,omitempty" bson:"activityId,omitempty"`
	Name            *string            `json:"name" bson:"name" example:"Quarter Final"`
	Description     *string            `json:"description" bson:"description" example:"Quarter Final"`
	MaxParticipants *int               `json:"maxParticipants" bson:"maxParticipants" example:"22"`
	Room            *string            `json:"room" bson:"room" example:"Stadium A"`
	Operator        *string            `json:"operator" bson:"operator" example:"Operator 1"`
	Dates           []Dates            `json:"dates" bson:"dates"`
	Hour            *int               `json:"hour" bson:"hour"  example:"4"`
	Enrollments     []Enrollment       `json:"enrollments" `
}

type Dates struct {
	Date  string `json:"date" bson:"date" example:"2025-03-11"`
	Stime string `json:"stime" bson:"stime" example:"10:00"`
	Etime string `json:"etime" bson:"etime" example:"12:00"`
}

// validate ยังไม่ถูกใช้งาน เพราะยังไม่อยากให้มันยังไม่ติด validate

// RequestCreateActivity ใช้สำหรับ CreateActivity API
type ActivityDto struct {
	ID            primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name          *string            `json:"name"`
	Type          string             `json:"type" `
	ActivityState string             `json:"activityState"  example:"planning"`
	Skill         string             `json:"skill"  example:"hard"`
	File          string             `json:"file"  example:"image.jpg"`
	StudentYears  []int              `json:"studentYears" example:"1,2,3,4"`
	Majors        []Major            `json:"majors"`
	ActivityItems []ActivityItem     `json:"activityItems"`
}

type EnrollmentSummary struct {
	MaxParticipants  int               `json:"maxParticipants"`
	TotalRegistered  int               `json:"totalRegistered"`
	RemainingSlots   int               `json:"remainingSlots"`
	ActivityItemSums []ActivityItemSum `json:"activityItemSums"`
}

type ActivityItemSum struct {
	ActivityItemName  string            `json:"activityItemName"`
	RegisteredByMajor []MajorEnrollment `json:"registeredByMajor"`
}

// โครงสร้างสำหรับแยกจำนวนลงทะเบียนตามสาขา
type MajorEnrollment struct {
	MajorName string `json:"majorName"`
	Count     int    `json:"count"`
}
