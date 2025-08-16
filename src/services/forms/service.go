package services

import (
	"context"
	"errors"

	"Backend-Bluelock-007/src/database" // สมมติว่าเชื่อมต่อ DB ไว้ที่นี่
	"Backend-Bluelock-007/src/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrInvalidObjectID = errors.New("invalid objectid")

// สร้างฟอร์ม
func InsetForm(ctx context.Context, form *models.Form) (*mongo.InsertOneResult, error) {
	return database.FormCollection.InsertOne(ctx, form)
}

// ดึงฟอร์มทั้งหมด
func GetAllForms(ctx context.Context) ([]models.Form, error) {
	cursor, err := database.FormCollection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var forms []models.Form
	if err := cursor.All(ctx, &forms); err != nil {
		return nil, err
	}
	return forms, nil
}

// ลบฟอร์มตาม ObjectID
func DeleteFormByID(ctx context.Context, id string) (*mongo.DeleteResult, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"_id": objID}
	return database.FormCollection.DeleteOne(ctx, filter)
}

// ดึงฟอร์มตาม ObjectID
func GetFormByID(ctx context.Context, id string) (*models.Form, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidObjectID
	}

	filter := bson.M{"_id": objID}
	var form models.Form
	if err := database.FormCollection.FindOne(ctx, filter).Decode(&form); err != nil {
		// ส่งต่อให้ controller แยกแยะ NotFound ด้วย mongo.ErrNoDocuments
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, mongo.ErrNoDocuments
		}
		return nil, err
	}
	return &form, nil
}


// UpdateForm updates an existing form by ID
func UpdateForm(ctx context.Context, id string, form *models.Form) (*mongo.UpdateResult, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidObjectID
	}

	// เตรียม $set เฉพาะฟิลด์ที่ต้องอัปเดต
	set := bson.M{
		"title":       form.Title,
		"description": form.Description,
		"activityId":  form.ActivityID,
		"isOrigin":    form.IsOrigin,
		"blocks":      form.Blocks,
	}

	result, err := database.FormCollection.UpdateByID(ctx, objID, bson.M{"$set": set})
	if err != nil {
		return nil, err
	}

	return result, nil
}


