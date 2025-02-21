package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User - โครงสร้างข้อมูลของผู้ใช้
type Activity struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	Name            string             `bson:"name"`
	Type            string             `bson:"type"`
	MajorID         int                `bson:"majorId"`
	AdminID         int                `bson:"adminId"`
	ActivityStateID int                `bson:"activityStateId"`
	SkillID         int                `bson:"skillId"`
	ActivityItem    []ActivityItem     `bson:"activityItems,omitempty"`
}

type ActivityItem struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	Name            string             `bson:"name"`
	MaxParticipants int                `bson:"maxParticipants"`
	Description     string             `bson:"description"`
	Room            string             `bson:"room"`
	StartDate       time.Time          `bson:"startDate"`
	EndDate         time.Time          `bson:"endDate"`
	Duration        int                `bson:"duration"`
	Operator        string             `bson:"operator"`
	Hour            int                `bson:"hour"`
	ActivityID      primitive.ObjectID `bson:"activityId"`
}
