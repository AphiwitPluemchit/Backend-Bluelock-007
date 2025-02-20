package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Major สาขาวิชา
type Major struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	MajorName string             `bson:"majorName"`
}
