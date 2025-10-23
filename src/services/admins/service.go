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
	"golang.org/x/crypto/bcrypt"
)

// Collections are now initialized in service.go

func GetAllAdmins(params models.PaginationParams) ([]bson.M, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	page := params.Page
	if page < 1 {
		page = 1
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}

	sortField := params.SortBy
	if sortField == "" {
		sortField = "_id" // ใช้ _id ใน Mongo
	}
	sortOrder := 1
	if strings.ToLower(params.Order) == "desc" {
		sortOrder = -1
	}

	pipeline := mongo.Pipeline{
		// เชื่อม Users เพื่อดึง email (Users.refId => Admins._id)
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "Users",
			"localField":   "_id",
			"foreignField": "refId",
			"as":           "user",
		}}},
	}

	// ค้นหา: name (ใน Admin) + user.email (ใน Users)
	if s := strings.TrimSpace(params.Search); s != "" {
		reg := bson.M{"$regex": s, "$options": "i"}
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{
			"$or": bson.A{
				bson.M{"name": reg},
				bson.M{"user.email": reg},
			},
		}}})
	}

	// addFields: email = ตัวแรกของ user.email
	pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
		"email": bson.M{"$arrayElemAt": bson.A{"$user.email", 0}}, // ใช้ index 0
	}}})

	// --- ทำ count ก่อนแบ่งหน้า ---
	countPipeline := append(mongo.Pipeline{}, pipeline...)
	countPipeline = append(countPipeline, bson.D{{Key: "$count", Value: "total"}})

	var total int64
	countCur, err := DB.AdminCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer countCur.Close(ctx)

	if countCur.Next(ctx) {
		var cr struct {
			Total int64 `bson:"total"`
		}
		if err := countCur.Decode(&cr); err == nil {
			total = cr.Total
		}
	}
	if total == 0 {
		return []bson.M{}, 0, 0, nil
	}

	// --- เรียง/แบ่งหน้า + project ฟิลด์ที่จะส่งออก ---
	mainPipeline := append(mongo.Pipeline{}, pipeline...)
	mainPipeline = append(mainPipeline,
		bson.D{{Key: "$sort", Value: bson.M{sortField: sortOrder}}},
		bson.D{{Key: "$skip", Value: int64((page - 1) * limit)}},
		bson.D{{Key: "$limit", Value: int64(limit)}},
		bson.D{{Key: "$project", Value: bson.M{
			"_id":   0,
			"id":    "$_id",
			"name":  1,
			"role":  1, // เอาออกได้ถ้าไม่ใช้
			"email": 1,
		}}},
	)

	cur, err := DB.AdminCollection.Aggregate(ctx, mainPipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cur.Close(ctx)

	var results []bson.M
	if err := cur.All(ctx, &results); err != nil {
		return nil, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return results, total, totalPages, nil
}

func GetAdminByID(id string) (bson.M, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("ไม่พบผู้ดูแล")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"_id": objID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "Users",
			"localField":   "_id",
			"foreignField": "refId",
			"as":           "user",
		}}},
		{{Key: "$addFields", Value: bson.M{
			"email": bson.M{"$arrayElemAt": bson.A{"$user.email", 0}}, // ดึง email ตัวแรก
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":   1,
			"name":  1,
			"email": 1,
			// เพิ่มฟิลด์อื่นที่อยากส่งออกได้ตรงนี้
		}}},
	}

	cur, err := DB.AdminCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	if !cur.Next(ctx) {
		return nil, errors.New("admin not found")
	}

	var doc bson.M
	if err := cur.Decode(&doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func CreateAdmin(userInput *models.User, adminInput *models.Admin) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ✅ ตรวจสอบว่า email ซ้ำใน Users หรือไม่
	count, err := DB.UserCollection.CountDocuments(ctx, bson.M{"email": userInput.Email})
	if err != nil {
		log.Println("❌ Error checking duplicate email:", err)
		return errors.New("failed to check duplicate email")
	}
	if count > 0 {
		return errors.New("มี email นี้ในระบบแล้ว")
	}

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
		return errors.New("สร้างข้อมูลไม่สำเร็จ")
	}

	// ✅ สร้าง user โดยใช้ refId อ้างถึง admin
	userInput.ID = primitive.NewObjectID()
	userInput.Role = "Admin"
	userInput.RefID = adminInput.ID
	userInput.IsActive = true

	_, err = DB.UserCollection.InsertOne(ctx, userInput)
	if err != nil {
		// rollback admin ถ้า user สร้างไม่สำเร็จ
		_, _ = DB.AdminCollection.DeleteOne(ctx, bson.M{"_id": adminInput.ID})
		return errors.New("สร้างข้อมูลไม่สำเร็จ")
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
		return errors.New("ไม่พบผู้ดูแล")
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
