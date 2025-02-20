package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivityItem กิจกรรมย่อย
type ActivityItem struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	Name            string             `bson:"name"`
	MaxParticipants int                `bson:"maxParticipants"`
	Description     string             `bson:"description"`
	Room            string             `bson:"room"`
	StartDate       time.Time          `bson:"startDate"`
	EndDate         time.Time          `bson:"endDate"`
	Duration        int                `bson:"duration"`
	Operator        string             `bson:"operator"`
	Hour            int                `bson:"hour"`
	ActivityID      primitive.ObjectID `bson:"activityId"`
}
