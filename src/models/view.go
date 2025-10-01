package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Summary_Check_In_Out_Reports struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProgramID        primitive.ObjectID `bson:"programId" json:"programId"`
	Date             string             `json:"date" bson:"date" example:"2025-03-11"`
	Registered       int                `bson:"registered" json:"registered"`
	Checkin          int                `bson:"checkin" json:"checkin"`
	CheckinLate      int                `bson:"checkinLate" json:"checkinLate"`
	Checkout         int                `bson:"checkout" json:"checkout"`
	NotParticipating int                `bson:"notParticipating" json:"notParticipating"`
}
