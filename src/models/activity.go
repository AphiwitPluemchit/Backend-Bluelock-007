package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Activity struct {
	ID              primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name            string             `json:"name" bson:"name"`
	Type            string             `json:"type" bson:"type"`
	AdminID         string             `json:"adminId" bson:"adminId"`                 // รับเป็น string
	ActivityStateID string             `json:"activityStateId" bson:"activityStateId"` // รับเป็น string
	SkillID         string             `json:"skillId" bson:"skillId"`                 // รับเป็น string
	MajorIDs        []string           `json:"majorIds" bson:"majorIds"`               // รับเป็น []string
	ActivityItems   []ActivityItem     `json:"activityItems" bson:"activityItems"`
}

type ActivityItem struct {
	ID              primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ActivityID      primitive.ObjectID `json:"activityId,omitempty" bson:"activityId,omitempty"`
	Name            string             `json:"name" bson:"name"`
	MaxParticipants int                `json:"maxParticipants" bson:"maxParticipants"`
	Description     string             `json:"description" bson:"description"`
	Room            string             `json:"room" bson:"room"`
	StartDate       string             `json:"startDate" bson:"startDate"`
	EndDate         string             `json:"endDate" bson:"endDate"`
	Duration        int                `json:"duration" bson:"duration"`
	Operator        string             `json:"operator" bson:"operator"`
	Hour            int                `json:"hour" bson:"hour"`
}
