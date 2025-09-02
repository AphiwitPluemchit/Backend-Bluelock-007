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
	NameMatch       int                `bson:"nameMatch" json:"nameMatch"`
	CourseMatch     int                `bson:"courseMatch" json:"courseMatch"`
	Status          StatusType         `bson:"status" json:"status" default:"pending" enum:"pending,approved,rejected"`
	Remark          string             `bson:"remark" json:"remark"`
	IsDuplicate     bool               `bson:"isDuplicate" json:"isDuplicate" default:"false"`
	UploadAt        time.Time          `bson:"uploadAt" json:"uploadAt" default:"time.Now()"`
	ChangedStatusAt *time.Time         `bson:"changedStatusAt" json:"changedStatusAt"`
}

type StatusType string

const (
	StatusPending  StatusType = "pending"
	StatusApproved StatusType = "approved"
	StatusRejected StatusType = "rejected"
)

type VerifyURLRequest struct {
	URL       string `query:"url" example:"https://learner.thaimooc.ac.th/credential-wallet/10793bb5-6e4f-4873-9309-f25f216a46c7/sahaphap.rit/public"`
	StudentID string `query:"studentId" example:"685abb936c4acf57c7e2e6ee"`
	CourseID  string `query:"courseId" example:"6890a82eebc423e6aeb56057"`
}
