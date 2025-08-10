package services

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

func CreateUploadCertificate(uploadCertificate *models.UploadCertificate) (*mongo.InsertOneResult, error) {

	return DB.UploadCertificateCollection.InsertOne(context.Background(), uploadCertificate)
}
