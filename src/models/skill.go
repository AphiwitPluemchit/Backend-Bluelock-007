package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Skill ทักษะ
type Skill struct {
	ID   primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`
}
