package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Food อาหาร
type Food struct {
	ID   primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`
}
type CreateFoodInput struct {
	Name string `json:"name" bson:"name"`
}