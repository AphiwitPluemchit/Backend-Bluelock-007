package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Admin เจ้าหน้าที่
type Admin struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name string             `bson:"name" json:"name"`
}
