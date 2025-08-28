package submission

import (
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"Backend-Bluelock-007/src/models"
)

type SubmissionService struct {
	collection *mongo.Collection
}

func NewSubmissionService(db *mongo.Database) *SubmissionService {
	return &SubmissionService{
		collection: db.Collection("submissions"),
	}
}

// CreateSubmission creates a new form submission
func (s *SubmissionService) CreateSubmission(ctx context.Context, submission *models.Submission) (*models.Submission, error) {
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
	res, err := s.collection.InsertOne(ctx, submission)
	if err != nil {
		return nil, err
	}

	// sync inserted id (เผื่อไดรเวอร์คืนค่า id ใหม่)
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		submission.ID = oid
	}

	log.Printf("[submission] inserted id=%s db=%s coll=%s responses=%d",
		submission.ID.Hex(), s.collection.Database().Name(), s.collection.Name(), len(submission.Responses))

	return submission, nil
}

// GetSubmissionByID retrieves a submission by its ID
func (s *SubmissionService) GetSubmissionByID(ctx context.Context, id primitive.ObjectID) (*models.Submission, error) {
	var submission models.Submission
	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&submission)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("submission not found")
		}
		return nil, err
	}

	return &submission, nil
}

// GetSubmissionsByFormID retrieves all submissions for a specific form
func (s *SubmissionService) GetSubmissionsByFormID(ctx context.Context, formID primitive.ObjectID) ([]models.Submission, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"formId": formID})
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
func (s *SubmissionService) GetSubmissionsByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Submission, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"userId": userID})
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
func (s *SubmissionService) DeleteSubmission(ctx context.Context, id primitive.ObjectID) error {
	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("no submission found with the given ID")
	}

	return nil
}
