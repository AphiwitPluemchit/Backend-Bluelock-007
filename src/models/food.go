package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Food อาหาร
type Food struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Name string             `bson:"name"`
}
