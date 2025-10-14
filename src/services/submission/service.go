package submission

import (
	DB "Backend-Bluelock-007/src/database"
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"Backend-Bluelock-007/src/models"
)
type BlockCountItem struct {
	BlockID  string `json:"blockId"  bson:"blockId"`
	ChoiceID string `json:"choiceId,omitempty" bson:"choiceId,omitempty"`
	RowID    string `json:"rowId,omitempty"    bson:"rowId,omitempty"`
	Count    int64  `json:"count"    bson:"count"`
}
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




func GetSubmissionsByFormID(ctx context.Context, formID primitive.ObjectID, limit int64, sortField string) ([]models.Submission, error) {
  filter := bson.M{"formId": formID}

  findOpts := options.Find()
  if limit > 0 {
    findOpts.SetLimit(limit)
  }
  if sortField != "" {
    sort := bson.D{}
    if strings.HasPrefix(sortField, "-") {
      sort = append(sort, bson.E{Key: strings.TrimPrefix(sortField, "-"), Value: -1})
    } else {
      sort = append(sort, bson.E{Key: sortField, Value: 1})
    }
    findOpts.SetSort(sort)
  } else {
    // ค่าเริ่มต้น: ใหม่สุดก่อน
    findOpts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
  }

  cursor, err := DB.SubmissionCollection.Find(ctx, filter, findOpts)
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
func GetFormBlocksAnalytics(ctx context.Context, formID primitive.ObjectID) ([]BlockCountItem, error) {
	pipeline := []bson.M{
			{"$match": bson.M{"formId": formID}},
			{"$unwind": "$responses"},
			{"$group": bson.M{
					"_id": bson.M{
							"blockId":  "$responses.blockId",
							"choiceId": "$responses.choiceId",
							"rowId":    "$responses.rowId",
					},
					"count": bson.M{"$sum": 1},
			}},
			{"$project": bson.M{
					"blockId": bson.M{"$toString": "$_id.blockId"},
					"choiceId": bson.M{
							"$ifNull": []interface{}{bson.M{"$toString": "$_id.choiceId"}, ""},
					},
					"rowId": bson.M{
							"$ifNull": []interface{}{bson.M{"$toString": "$_id.rowId"}, ""},
					},
					"count": 1,
			}},
	}

	cur, err := DB.SubmissionCollection.Aggregate(ctx, pipeline)
	if err != nil {
			return nil, err
	}
	defer cur.Close(ctx)

	var out []BlockCountItem
	if err := cur.All(ctx, &out); err != nil {
			return nil, err
	}
	return out, nil
}

// รายบล็อก
func GetBlockAnalytics(ctx context.Context, formID, blockID primitive.ObjectID) ([]BlockCountItem, error) {
	pipeline := []bson.M{
			{"$match": bson.M{"formId": formID}},
			{"$unwind": "$responses"},
			{"$match": bson.M{"responses.blockId": blockID}},
			{"$group": bson.M{
					"_id": bson.M{
							"blockId":  "$responses.blockId",
							"choiceId": "$responses.choiceId",
							"rowId":    "$responses.rowId",
					},
					"count": bson.M{"$sum": 1},
			}},
			{"$project": bson.M{
					"blockId": bson.M{"$toString": "$_id.blockId"},
					"choiceId": bson.M{
							"$ifNull": []interface{}{bson.M{"$toString": "$_id.choiceId"}, ""},
					},
					"rowId": bson.M{
							"$ifNull": []interface{}{bson.M{"$toString": "$_id.rowId"}, ""},
					},
					"count": 1,
			}},
	}

	cur, err := DB.SubmissionCollection.Aggregate(ctx, pipeline)
	if err != nil {
			return nil, err
	}
	defer cur.Close(ctx)

	var out []BlockCountItem
	if err := cur.All(ctx, &out); err != nil {
			return nil, err
	}
	return out, nil
}