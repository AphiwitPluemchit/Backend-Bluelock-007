package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UploadCertificate struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty" `
	StudentId       primitive.ObjectID `bson:"studentId" json:"studentId"`
	CourseId        primitive.ObjectID `bson:"courseId" json:"courseId"`
	Url             string             `bson:"url" json:"url"`
	IsNameMatch     bool               `bson:"isNameMatch" json:"isNameMatch"`
	IsCourseMatch   bool               `bson:"isCourseMatch" json:"isCourseMatch"`
	Status          string             `bson:"status" json:"status" default:"pending" enum:"pending,approved,rejected"`
	Remark          string             `bson:"remark" json:"remark"`
	IsDuplicate     bool               `bson:"isDuplicate" json:"isDuplicate" default:"false"`
	UploadAt        time.Time          `bson:"uploadAt" json:"uploadAt" default:"time.Now()"`
	ChangedStatusAt *time.Time         `bson:"changedStatusAt" json:"changedStatusAt"`
}
