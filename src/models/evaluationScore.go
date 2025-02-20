package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EvaluationScore คะแนนการประเมิน
type EvaluationScore struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	Score            int                `bson:"score"`
	CheckInOutID     primitive.ObjectID `bson:"checkInOutId"`
	FormEvaluationID primitive.ObjectID `bson:"formEvaluationId"`
}
