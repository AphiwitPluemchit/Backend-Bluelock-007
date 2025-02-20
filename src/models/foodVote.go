package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FoodVote โหวตอาหาร

type FoodVote struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Score      int                `bson:"score"`
	ActivityID primitive.ObjectID `bson:"activityId"`
	FoodID     primitive.ObjectID `bson:"foodId"`
}
