package admins

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"

	"context"
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// Collections are now initialized in service.go

func GetAllAdmins(params models.PaginationParams) ([]models.Admin, int64, int, error) {
	var admins []models.Admin
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skip := int64((params.Page - 1) * params.Limit)
	sortField := params.SortBy
	if sortField == "" {
		sortField = "id"
	}
	sortOrder := 1
	if strings.ToLower(params.Order) == "desc" {
		sortOrder = -1
	}

	filter := bson.M{}
	if params.Search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": params.Search, "$options": "i"}},
			{"email": bson.M{"$regex": params.Search, "$options": "i"}},
		}
	}
	total, err := DB.AdminCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	findOptions := options.Find().
		SetSkip(skip).
		SetLimit(int64(params.Limit)).
		SetSort(bson.D{{Key: sortField, Value: sortOrder}})

	cursor, err := DB.AdminCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var admin models.Admin
		if err := cursor.Decode(&admin); err != nil {
			log.Println("Error decoding admin:", err)
			continue
		}
		admins = append(admins, admin)
	}
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return admins, total, totalPages, nil
}

func GetAdminByID(id string) (*models.Admin, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid admin ID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admin models.Admin
	err = DB.AdminCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&admin)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errors.New("admin not found")
	} else if err != nil {
		log.Println("❌ Error finding admin:", err)
		return nil, err
	}
	return &admin, nil
}

func CreateAdmin(userInput *models.User, adminInput *models.Admin) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ✅ เข้ารหัสรหัสผ่าน
	hashedPassword, err := hashPassword(userInput.Password)
	if err != nil {
		return errors.New("failed to hash password")
	}
	userInput.Password = hashedPassword

	// ✅ สร้าง admin profile
	adminInput.ID = primitive.NewObjectID()
	_, err = DB.AdminCollection.InsertOne(ctx, adminInput)
	if err != nil {
		log.Println("❌ Error inserting admin:", err)
		return errors.New("failed to insert admin profile")
	}

	// ✅ สร้าง user โดยใช้ refId อ้างถึง admin
	userInput.ID = primitive.NewObjectID()
	userInput.Role = "Admin"
	userInput.RefID = adminInput.ID
	userInput.IsActive = true

	_, err = DB.UserCollection.InsertOne(ctx, userInput)
	if err != nil {
		DB.AdminCollection.DeleteOne(ctx, bson.M{"_id": adminInput.ID}) // rollback
		return errors.New("failed to create user for admin")
	}

	return nil
}

func UpdateAdmin(id string, admin *models.Admin) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid admin ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": admin}
	_, err = DB.AdminCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return err
	}

	// ✅ sync ชื่อใน users ด้วย
	_, err = DB.UserCollection.UpdateOne(context.Background(), bson.M{
		"refId": objID,
		"role":  "Admin",
	}, bson.M{
		"$set": bson.M{"name": admin.Name},
	})

	return err
}

func DeleteAdmin(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid admin ID")
	}
	// 🔧 ควรลบจาก user โดยใช้ refId ไม่ใช่ _id
	_, err = DB.UserCollection.DeleteOne(context.Background(), bson.M{
		"refId": objID,
		"role":  "Admin",
	})
	if err != nil {
		return err
	}

	_, err = DB.AdminCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
