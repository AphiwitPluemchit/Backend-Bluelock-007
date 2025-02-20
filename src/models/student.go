package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Student นิสิต
type Student struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Code      string             `bson:"code"`
	Name      string             `bson:"name"`
	Email     string             `bson:"email"`
	Status    string             `bson:"status"`
	Password  string             `bson:"password"`
	SoftSkill int                `bson:"softSkill"`
	HardSkill int                `bson:"hardSkill"`
	MajorID   int                `bson:"majorId"`
}
