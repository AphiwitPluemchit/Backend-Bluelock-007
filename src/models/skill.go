package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Skill ทักษะ
type Skill struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Name string             `bson:"name"`
}
