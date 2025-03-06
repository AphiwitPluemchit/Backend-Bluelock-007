package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivityState สถานะกิจกรรม
type ActivityState struct {
	ID   primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`
}
