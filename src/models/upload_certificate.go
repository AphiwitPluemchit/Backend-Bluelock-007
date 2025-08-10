package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type UploadCertificate struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty" `
	StudentId     primitive.ObjectID `bson:"studentId" json:"studentId"`
	CourseId      primitive.ObjectID `bson:"courseId" json:"courseId"`
	Url           string             `bson:"url" json:"url"`
	FileName      string             `bson:"fileName" json:"fileName"`
	IsNameMatch   bool               `bson:"isNameMatch" json:"isNameMatch"`
	IsCourseMatch bool               `bson:"isCourseMatch" json:"isCourseMatch"`
}
