package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Student นิสิต
type Student struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code      string             `bson:"code" json:"code"`
	Name      string             `bson:"name" json:"name"`
	EngName   string             `bson:"engName" json:"engName"`
	Status    int                `bson:"status" json:"status"`
	SoftSkill int                `bson:"softSkill" json:"softSkill"`
	HardSkill int                `bson:"hardSkill" json:"hardSkill"`
	Major     string             `bson:"major" json:"major"`
}
