package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Student นิสิต
type Student struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Code      string             `bson:"code"`
	Name      string             `bson:"name"`
	Email     string             `bson:"email,omitempty"`
	Status    string             `bson:"status"`
	Password  string             `bson:"password,omitempty"`
	SoftSkill int                `bson:"softSkill,omitempty"`
	HardSkill int                `bson:"hardSkill,omitempty"`
	MajorID   primitive.ObjectID `bson:"majorId"`
}
