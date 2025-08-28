package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Submission struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FormID    primitive.ObjectID `bson:"formId,omitempty" json:"formId"`
	UserID    primitive.ObjectID `bson:"userId,omitempty" json:"userId"`
	Responses []Response         `bson:"responses,omitempty" json:"responses"`
	CreatedAt time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

type Response struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	AnswerText *string             `bson:"answerText,omitempty" json:"answerText"`
	BlockID    primitive.ObjectID  `bson:"blockId,omitempty" json:"blockId"`
	ChoiceID   *primitive.ObjectID `bson:"choiceId,omitempty" json:"choiceId"` 
	RowID      *primitive.ObjectID `bson:"rowId,omitempty" json:"rowId"`   
}
