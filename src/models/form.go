package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// --- Form ---
type Form struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Title      string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	IsOrigin   bool               `bson:"isOrigin" json:"isOrigin"`
	// Category   string             `bson:"category" json:"category"`

	Blocks []Block `bson:"blocks,omitempty" json:"blocks,omitempty"`
}

// --- Block ---
type Block struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Title       string             `bson:"title" json:"title"`
	Session     int                `bson:"session" json:"session"`
	Type        string             `bson:"type" json:"type"`
	Description string             `bson:"description" json:"description"`
	IsRequired  bool               `bson:"isRequired" json:"isRequired"`
	Sequence    int                `bson:"sequence" json:"sequence"`
	FormID      primitive.ObjectID `bson:"formId" json:"formId"`

	Choices []Choice `bson:"choices,omitempty" json:"choices,omitempty"`
	Rows    []Row    `bson:"rows,omitempty" json:"rows,omitempty"`

	Max  *int    `bson:"max,omitempty" json:"max,omitempty"`   
	Icon *string `bson:"icon,omitempty" json:"icon,omitempty"` 
}

// --- Choice ---
type Choice struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Title    string             `bson:"title" json:"title"`
	Sequence int                `bson:"sequence" json:"sequence"`
	BlockID  primitive.ObjectID `bson:"blockId" json:"blockId"`
}

// --- Row ---
type Row struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Title    string             `bson:"title" json:"title"`
	Sequence int                `bson:"sequence" json:"sequence"`
	BlockID  primitive.ObjectID `bson:"blockId" json:"blockId"`
}

// --- Response ---
type Response struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"` 
	AnswerText *string            `bson:"answerText,omitempty" json:"answerText,omitempty"`
	BlockID    primitive.ObjectID `bson:"blockId" json:"blockId"`
	ChoiceID   *primitive.ObjectID `bson:"choiceId,omitempty" json:"choiceId,omitempty"`
	RowID      *primitive.ObjectID `bson:"rowId,omitempty" json:"rowId,omitempty"`
	UserID     primitive.ObjectID `bson:"userId" json:"userId"`
}
