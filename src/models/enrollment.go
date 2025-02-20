package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Enrollment การลงทะเบียน
type Enrollment struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	RegistrationDate time.Time          `bson:"registrationDate"`
	Food             string             `bson:"food"`
}
