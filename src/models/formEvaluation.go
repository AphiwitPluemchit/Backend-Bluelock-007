package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FormEvaluation แบบฟอร์มประเมิน
type FormEvaluation struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Name string             `bson:"name"`
}
