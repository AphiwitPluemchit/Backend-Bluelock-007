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

// ===================== CRUD พื้นฐาน =====================

func CreateSubmission(ctx context.Context, submission *models.Submission) (*models.Submission, error) {
	if submission.FormID.IsZero() {
		return nil, errors.New("form ID is required")
	}
	if submission.UserID.IsZero() {
		return nil, errors.New("user ID is required")
	}

	submission.ID = primitive.NewObjectID()
	submission.CreatedAt = time.Now()
	submission.UpdatedAt = time.Now()

	for i := range submission.Responses {
		if submission.Responses[i].ID.IsZero() {
			submission.Responses[i].ID = primitive.NewObjectID()
		}
	}

	res, err := DB.SubmissionCollection.InsertOne(ctx, submission)
	if err != nil {
		return nil, err
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		submission.ID = oid
	}

	log.Printf("[submission] inserted id=%s db=%s coll=%s responses=%d",
		submission.ID.Hex(), DB.SubmissionCollection.Database().Name(), DB.SubmissionCollection.Name(), len(submission.Responses))

	return submission, nil
}

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

func GetSubmissionsByFormIDWithQuery(ctx context.Context, formID primitive.ObjectID, limit int, sortParam string) ([]models.Submission, error) {
	filter := bson.M{"formId": formID}
	findOpts := options.Find()

	if strings.EqualFold(sortParam, "latest") {
		findOpts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
	}
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
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

// ===================== Analytics =====================

// โครงสร้างสำหรับอ่านค่าดิบจาก aggregate (ObjectID)
type blockCountRaw struct {
	BlockID  primitive.ObjectID  `bson:"blockId"`
	ChoiceID *primitive.ObjectID `bson:"choiceId,omitempty"`
	RowID    *primitive.ObjectID `bson:"rowId,omitempty"`
	Count    int64               `bson:"count"`
}

// โครงสร้างส่งออกให้ frontend (string)
type BlockCountDTO struct {
	BlockID  string  `json:"blockId"`
	ChoiceID *string `json:"choiceId,omitempty"`
	RowID    *string `json:"rowId,omitempty"`
	Count    int64   `json:"count"`
}

func toDTOs(in []blockCountRaw) []BlockCountDTO {
	out := make([]BlockCountDTO, 0, len(in))
	for _, r := range in {
		dto := BlockCountDTO{
			BlockID: r.BlockID.Hex(),
			Count:   r.Count,
		}
		if r.ChoiceID != nil {
			s := r.ChoiceID.Hex()
			dto.ChoiceID = &s
		}
		if r.RowID != nil {
			s := r.RowID.Hex()
			dto.RowID = &s
		}
		out = append(out, dto)
	}
	return out
}

// นับต่อ choice (และ row สำหรับ grid) เฉพาะ "บล็อกเดียว"
func AggregateBlockCounts(ctx context.Context, formID, blockID primitive.ObjectID) ([]BlockCountDTO, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"formId": formID}}},
		{{Key: "$unwind", Value: "$responses"}},
		// สำคัญ: กรองเฉพาะบล็อกนี้
		{{Key: "$match", Value: bson.M{"responses.blockId": blockID}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"blockId":  "$responses.blockId",
				"choiceId": "$responses.choiceId",
				"rowId":    "$responses.rowId",
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":     0,
			"blockId": "$_id.blockId",
			"choiceId": "$_id.choiceId",
			"rowId":   "$_id.rowId",
			"count":   1,
		}}},
	}

	cur, err := DB.SubmissionCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var raw []blockCountRaw
	if err := cur.All(ctx, &raw); err != nil {
		return nil, err
	}
	return toDTOs(raw), nil
}

// นับรวมทุกบล็อกในฟอร์ม
func AggregateFormCounts(ctx context.Context, formID primitive.ObjectID) ([]BlockCountDTO, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"formId": formID}}},
		{{Key: "$unwind", Value: "$responses"}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"blockId":  "$responses.blockId",
				"choiceId": "$responses.choiceId",
				"rowId":    "$responses.rowId",
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":     0,
			"blockId": "$_id.blockId",
			"choiceId": "$_id.choiceId",
			"rowId":   "$_id.rowId",
			"count":   1,
		}}},
	}

	cur, err := DB.SubmissionCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var raw []blockCountRaw
	if err := cur.All(ctx, &raw); err != nil {
		return nil, err
	}
	return toDTOs(raw), nil
}
