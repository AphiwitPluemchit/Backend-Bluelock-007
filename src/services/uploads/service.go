package services

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateUploadCertificate(uploadCertificate *models.UploadCertificate) (*mongo.InsertOneResult, error) {
	ctx := context.Background()
	return DB.UploadCertificateCollection.InsertOne(ctx, uploadCertificate)
}

func UpdateUploadCertificate(id string, uploadCertificate *models.UploadCertificate) (*mongo.UpdateResult, error) {
	ctx := context.Background()
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid upload certificate ID")
	}
	return DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": uploadCertificate})
}

func IsVerifiedDuplicate(ctx context.Context, url string) (bool, error) {
	if url == "" {
		return false, nil
	}

	filter := bson.M{
		"url":           url,
		"isNameMatch":   true,
		"isCourseMatch": true,
	}

	err := DB.UploadCertificateCollection.FindOne(ctx, filter).Err()
	switch err {
	case nil:
		// พบแล้ว → เป็นซ้ำ
		return true, nil
	case mongo.ErrNoDocuments:
		// ไม่พบ → ไม่ซ้ำ
		return false, nil
	default:
		// error อื่นจาก DB
		return false, err
	}
}
