package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FoodVote โหวตอาหาร

type FoodVote struct {
	ID         primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Vote       int                `json:"vote" bson:"vote"`
	FoodName   string             `json:"foodName" bson:"foodName"`
	ActivityID primitive.ObjectID `json:"activityId" bson:"activityId"`
}
