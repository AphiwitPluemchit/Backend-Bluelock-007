package submission

import (
	DB "Backend-Bluelock-007/src/database"
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"Backend-Bluelock-007/src/models"
)

// CreateSubmission creates a new form submission
func CreateSubmission(ctx context.Context, submission *models.Submission) (*models.Submission, error) {
	// Validate required fields
	if submission.FormID.IsZero() {
		return nil, errors.New("form ID is required")
	}
	if submission.UserID.IsZero() {
		return nil, errors.New("user ID is required")
	}

	// Set timestamps
	submission.ID = primitive.NewObjectID()
	submission.CreatedAt = time.Now()
	submission.UpdatedAt = time.Now()

	// Ensure all responses have IDs
	for i := range submission.Responses {
		if submission.Responses[i].ID.IsZero() {
			submission.Responses[i].ID = primitive.NewObjectID()
		}
	}

	// Insert into database
	res, err := DB.SubmissionCollection.InsertOne(ctx, submission)
	if err != nil {
		return nil, err
	}

	// sync inserted id (เผื่อไดรเวอร์คืนค่า id ใหม่)
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		submission.ID = oid
	}

	log.Printf("[submission] inserted id=%s db=%s coll=%s responses=%d",
		submission.ID.Hex(), DB.SubmissionCollection.Database().Name(), DB.SubmissionCollection.Name(), len(submission.Responses))

	return submission, nil
}

// GetSubmissionByID retrieves a submission by its ID
func GetSubmissionByID(ctx context.Context, id primitive.ObjectID) (*models.Submission, error) {
	var submission models.Submission
	err := DB.SubmissionCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&submission)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("submission not found")
		}
		return nil, err
	}

	return &submission, nil
}

// GetSubmissionsByFormID retrieves all submissions for a specific form
func GetSubmissionsByFormID(ctx context.Context, formID primitive.ObjectID) ([]models.Submission, error) {
	cursor, err := DB.SubmissionCollection.Find(ctx, bson.M{"formId": formID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var submissions []models.Submission
	if err := cursor.All(ctx, &submissions); err != nil {
		return nil, err
	}

	return submissions, nil
}

// GetSubmissionsByUserID retrieves all submissions made by a specific user
func GetSubmissionsByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Submission, error) {
	cursor, err := DB.SubmissionCollection.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var submissions []models.Submission
	if err := cursor.All(ctx, &submissions); err != nil {
		return nil, err
	}

	return submissions, nil
}

// DeleteSubmission deletes a submission by its ID
func DeleteSubmission(ctx context.Context, id primitive.ObjectID) error {
	result, err := DB.SubmissionCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("no submission found with the given ID")
	}

	return nil
}
