package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Suggestion ข้อเสนอแนะ
type Suggestion struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	SuggestionText   string             `bson:"suggestion"`
	FormEvaluationID primitive.ObjectID `bson:"formEvaluationId"`
}
