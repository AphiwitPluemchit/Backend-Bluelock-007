package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Activity กิจกรรมหลัก
type Activity struct {
	ID            primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	FormID        primitive.ObjectID `json:"formId,omitempty" bson:"formId,omitempty"`
	Name          *string            `json:"name" bson:"name" example:"Football Tournament"`
	Type          string             `json:"type" bson:"type" example:"one"`
	ActivityState string             `json:"activityState" bson:"activityState" example:"planning"`
	Skill         string             `json:"skill" bson:"skill" example:"hard"`
	EndDateEnroll string             `json:"endDateEnroll" bson:"endDateEnroll"`
	File          string             `json:"file" bson:"file"  example:"image.jpg"`
	FoodVotes     []FoodVote         `json:"foodVotes" bson:"foodVotes"`
}

type ActivityDto struct {
	ID            primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	FormID        primitive.ObjectID `json:"formId,omitempty" bson:"formId,omitempty"`
	Name          *string            `json:"name" bson:"name" example:"Football Tournament"`
	Type          string             `json:"type" bson:"type" example:"one"`
	ActivityState string             `json:"activityState" bson:"activityState" example:"planning"`
	Skill         string             `json:"skill" bson:"skill" example:"hard"`
	EndDateEnroll string             `json:"endDateEnroll" bson:"endDateEnroll"`
	File          string             `json:"file" bson:"file"  example:"image.jpg"`
	FoodVotes     []FoodVote         `json:"foodVotes" bson:"foodVotes"`
	ActivityItems []ActivityItemDto  `json:"activityItems" `
}

// ActivityItem รายละเอียดกิจกรรมย่อย
type ActivityItem struct {
	ID              primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ActivityID      primitive.ObjectID `json:"activityId,omitempty" bson:"activityId,omitempty"`
	Name            *string            `json:"name" bson:"name" example:"Quarter Final"`
	Description     *string            `json:"description" bson:"description" example:"Quarter Final"`
	StudentYears    []int              `json:"studentYears" bson:"studentYears" example:"1,2,3,4"`
	MaxParticipants *int               `json:"maxParticipants" bson:"maxParticipants" example:"22"`
	Majors          []string           `json:"majors" bson:"majors" example:"CS,SE,ITDI,AAI"`
	Rooms           *[]string          `json:"rooms" bson:"rooms" example:"Room 1,Room 2"`
	Operator        *string            `json:"operator" bson:"operator" example:"Operator 1"`
	Dates           []Dates            `json:"dates" bson:"dates" `
	Hour            *int               `json:"hour" bson:"hour"  example:"4"`
	EnrollmentCount int                `json:"enrollmentCount"  `
}

type ActivityItemDto struct {
	ID              primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ActivityID      primitive.ObjectID `json:"activityId,omitempty" bson:"activityId,omitempty"`
	Name            *string            `json:"name" bson:"name" example:"Quarter Final"`
	Description     *string            `json:"description" bson:"description" example:"Quarter Final"`
	StudentYears    []int              `json:"studentYears" bson:"studentYears" example:"1,2,3,4"`
	MaxParticipants *int               `json:"maxParticipants" bson:"maxParticipants" example:"22"`
	Majors          []string           `json:"majors" bson:"majors" example:"CS,SE,ITDI,AAI"`
	Rooms           *[]string          `json:"rooms" bson:"rooms" example:"Room 1,Room 2"`
	Operator        *string            `json:"operator" bson:"operator" example:"Operator 1"`
	Dates           []Dates            `json:"dates" bson:"dates" `
	Hour            *int               `json:"hour" bson:"hour"  example:"4"`
	EnrollmentCount int                `json:"enrollmentCount"  `
}

type Dates struct {
	Date  string `json:"date" bson:"date" example:"2025-03-11"`
	Stime string `json:"stime" bson:"stime" example:"10:00"`
	Etime string `json:"etime" bson:"etime" example:"12:00"`
}

type FoodVote struct {
	Vote     int    `json:"vote" bson:"vote"`
	FoodName string `json:"foodName" bson:"foodName" example:"Pizza"`
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
	MajorName string `json:"majorName" `
	Count     int    `json:"count"`
}

type ActivityHistory struct {
	ID            primitive.ObjectID    `json:"id,omitempty" bson:"_id,omitempty"`
	Name          *string               `json:"name" bson:"name" example:"Football Tournament"`
	Skill         string                `json:"skill" bson:"skill" example:"hard"`
	File          string                `json:"file" bson:"file"  example:"image.jpg"`
	ActivityItems []ActivityItemHistory `json:"activityItems" `
}
type ActivityItemHistory struct {
	ID               primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ActivityID       primitive.ObjectID `json:"activityId,omitempty" bson:"activityId,omitempty"`
	Name             *string            `json:"name" bson:"name" example:"Quarter Final"`
	Description      *string            `json:"description" bson:"description" example:"Quarter Final"`
	StudentYears     []int              `json:"studentYears" bson:"studentYears" example:"1,2,3,4"`
	MaxParticipants  *int               `json:"maxParticipants" bson:"maxParticipants" example:"22"`
	Majors           []string           `json:"majors" bson:"majors" example:"CS,SE,ITDI,AAI"`
	Rooms            *[]string          `json:"rooms" bson:"rooms" example:"Room 1,Room 2"`
	Operator         *string            `json:"operator" bson:"operator" example:"Operator 1"`
	Dates            []Dates            `json:"dates" bson:"dates" `
	Hour             *int               `json:"hour" bson:"hour"  example:"4"`
	EnrollmentCount  int                `json:"enrollmentCount"  `
	CheckinoutRecord []CheckinoutRecord `json:"checkinoutRecord" `
}
