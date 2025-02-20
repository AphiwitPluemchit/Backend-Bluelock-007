package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// User - โครงสร้างข้อมูลของผู้ใช้
type Activity struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	Name            string             `bson:"name"`
	Type            string             `bson:"type"`
	MajorID         int                `bson:"majorId"`
	AdminID         int                `bson:"adminId"`
	ActivityStateID int                `bson:"activityStateId"`
	SkillID         int                `bson:"skillId"`
}
