package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FoodVote โหวตอาหาร

type FoodVote struct {
	ID         primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Vote       int                `json:"vote" bson:"vote"`
	ActivityID primitive.ObjectID `json:"activityId" bson:"activityId"`
	FoodID     primitive.ObjectID `json:"foodId" bson:"foodId"`
	Food       Food               `json:"food" bson:"-"` // ❌ ห้ามบันทึก Food ลง MongoDB
}
