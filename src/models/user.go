package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Admin เจ้าหน้าที่
type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email    string             `bson:"email" json:"email"`
	Password string             `bson:"password,omitempty" json:"-"` // ✅ ส่งมาได้จาก frontend, แต่ไม่ส่งกลับ
	Role     string             `bson:"role" json:"role"`
	RefID    primitive.ObjectID `bson:"refId" json:"refId"`
	Name     string             `bson:"-" json:"name"`
}
