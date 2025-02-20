package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivityState สถานะกิจกรรม
type ActivityState struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Name string             `bson:"name"`
}
