package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Major สาขาวิชา
type Major struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	MajorName string             ` json:"majorName" bson:"majorName"`
}
