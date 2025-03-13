package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Admin เจ้าหน้าที่
type User struct {
	ID        primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	Email     string              `json:"email" bson:"email"`
	Password  string              `json:"-" bson:"password"`
	Role      string              `json:"role" bson:"role"`
	StudentID *primitive.ObjectID `json:"studentId,omitempty" bson:"studentId,omitempty"`
	AdminID   *primitive.ObjectID `json:"adminId,omitempty" bson:"adminId,omitempty"`
}
