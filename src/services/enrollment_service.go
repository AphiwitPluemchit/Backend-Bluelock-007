package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var enrollmentCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	enrollmentCollection = database.GetCollection("BluelockDB", "enrollments")
	activityItemCollection = database.GetCollection("BluelockDB", "activityItems")
	studentCollection = database.GetCollection("BluelockDB", "students")

	if enrollmentCollection == nil || activityItemCollection == nil || studentCollection == nil {
		log.Fatal("Failed to get necessary collections")
	}
}

// ✅ 1. Student ลงทะเบียนกิจกรรม (ลงซ้ำไม่ได้)
func RegisterStudent(activityItemID, studentID primitive.ObjectID, food *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ✅ ตรวจสอบว่า ActivityItem และ Student มีอยู่จริงไหม
	var activityItem models.ActivityItem
	if err := activityItemCollection.FindOne(ctx, bson.M{"_id": activityItemID}).Decode(&activityItem); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("activity item not found")
		}
		return err
	}

	if food != nil {
		activityID := activityItem.ActivityID

		// ✅ Update +1 vote ของ foodName ที่ตรงกับชื่ออาหาร
		filter := bson.M{"_id": activityID}
		update := bson.M{
			"$inc": bson.M{"foodVotes.$[elem].vote": 1},
		}
		arrayFilter := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []any{
				bson.M{"elem.foodName": *food},
			},
		})

		// ✅ Run update
		_, err := activityCollection.UpdateOne(ctx, filter, update, arrayFilter)
		if err != nil {
			return err
		}

		fmt.Println("Updated food vote for:", *food)
	}

	// ✅ ตรวจสอบเวลาทับซ้อนก่อนลงทะเบียน
	existingEnrollmentsCursor, err := enrollmentCollection.Find(ctx, bson.M{"studentId": studentID})
	if err != nil {
		return err
	}
	defer existingEnrollmentsCursor.Close(ctx)

	for existingEnrollmentsCursor.Next(ctx) {
		var existing models.Enrollment
		if err := existingEnrollmentsCursor.Decode(&existing); err != nil {
			continue
		}

		// ดึง activityItem เดิมที่เคยลง
		var existingItem models.ActivityItem
		if err := activityItemCollection.FindOne(ctx, bson.M{"_id": existing.ActivityItemID}).Decode(&existingItem); err != nil {
			continue
		}

		// เปรียบเทียบวัน
		for _, dOld := range existingItem.Dates {
			for _, dNew := range activityItem.Dates {
				if dOld.Date == dNew.Date {
					// ✅ ถ้าวันตรงกัน ให้เช็คเวลาทับซ้อน
					if isTimeOverlap(dOld.Stime, dOld.Etime, dNew.Stime, dNew.Etime) {
						return errors.New("ไม่สามารถลงทะเบียนได้ เนื่องจากมีกิจกรรมที่เวลาเดียวกันอยู่แล้ว")
					}
				}
			}
		}
	}

	var student models.Student
	if err := studentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("student not found")
		}
		return err
	}

	// ✅ ตรวจสอบว่าลงทะเบียนไปแล้วหรือยัง
	count, err := enrollmentCollection.CountDocuments(ctx, bson.M{
		"activityItemId": activityItemID,
		"studentId":      studentID,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("already enrolled in this activity")
	}

	// ✅ สร้าง Enrollment ใหม่ พร้อม food ถ้ามี
	newEnrollment := models.Enrollment{
		ID:               primitive.NewObjectID(),
		StudentID:        studentID,
		ActivityItemID:   activityItemID,
		RegistrationDate: time.Now(),
		Food:             food, // ✅ เก็บค่าอาหารที่ส่งมา หรือ nil
	}

	_, err = enrollmentCollection.InsertOne(ctx, newEnrollment)
	return err
}

// ✅ 2. ดึงกิจกรรมทั้งหมดที่ Student ลงทะเบียนไปแล้ว พร้อม pagination และ filter
func GetEnrollmentsByStudent(studentID primitive.ObjectID, params models.PaginationParams, skillFilter []string) ([]models.Activity, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ✅ Step 1: ดึง activityItemIds จาก enrollment ที่ student ลงทะเบียน
	matchStage := bson.D{{Key: "$match", Value: bson.M{"studentId": studentID}}}
	lookupActivityItem := bson.D{{Key: "$lookup", Value: bson.M{
		"from":         "activityItems",
		"localField":   "activityItemId",
		"foreignField": "_id",
		"as":           "activityItemDetails",
	}}}
	unwindActivityItem := bson.D{{Key: "$unwind", Value: "$activityItemDetails"}}
	groupActivityIDs := bson.D{{Key: "$group", Value: bson.M{
		"_id":             nil,
		"activityItemIds": bson.M{"$addToSet": "$activityItemDetails._id"},
		"activityIds":     bson.M{"$addToSet": "$activityItemDetails.activityId"},
	}}}

	enrollmentStage := mongo.Pipeline{matchStage, lookupActivityItem, unwindActivityItem, groupActivityIDs}
	cur, err := enrollmentCollection.Aggregate(ctx, enrollmentStage)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error fetching enrollments: %v", err)
	}
	var enrollmentResult []bson.M
	if err := cur.All(ctx, &enrollmentResult); err != nil || len(enrollmentResult) == 0 {
		return []models.Activity{}, 0, 0, nil
	}
	activityIDs := enrollmentResult[0]["activityIds"].(primitive.A)

	// ✅ Step 2: Filter + Paginate + Lookup activities เหมือน GetAllActivities
	skip := int64((params.Page - 1) * params.Limit)
	sort := bson.D{{Key: params.SortBy, Value: 1}}
	if strings.ToLower(params.Order) == "desc" {
		sort[0].Value = -1
	}

	filter := bson.M{"_id": bson.M{"$in": activityIDs}}
	if params.Search != "" {
		filter["name"] = bson.M{"$regex": params.Search, "$options": "i"}
	}
	if len(skillFilter) > 0 && skillFilter[0] != "" {
		filter["skill"] = bson.M{"$in": skillFilter}
	}

	total, err := activityCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	pipeline := getActivitiesPipeline(filter, params.SortBy, sort[0].Value.(int), skip, int64(params.Limit), []string{}, []int{})
	cursor, err := activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var activities []models.Activity
	if err := cursor.All(ctx, &activities); err != nil {
		return nil, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return activities, total, totalPages, nil
}

// ✅ 3. ยกเลิกการลงทะเบียน
func UnregisterStudent(enrollmentID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": enrollmentID}

	// get enrollment
	var enrollment models.Enrollment
	err := enrollmentCollection.FindOne(ctx, filter).Decode(&enrollment)
	if err != nil {
		return err
	}

	var activityItem models.ActivityItem
	if err := activityItemCollection.FindOne(ctx, bson.M{"_id": enrollment.ActivityItemID}).Decode(&activityItem); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("activity item not found")
		}
		return err
	}

	if enrollment.Food != nil {
		activityID := activityItem.ActivityID

		// ✅ Update -1 vote ของ foodName ที่ตรงกับชื่ออาหาร
		filter := bson.M{"_id": activityID}
		update := bson.M{
			"$inc": bson.M{"foodVotes.$[elem].vote": -1},
		}
		arrayFilter := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []any{
				bson.M{"elem.foodName": *enrollment.Food},
			},
		})

		// ✅ Run update
		_, err := activityCollection.UpdateOne(ctx, filter, update, arrayFilter)
		if err != nil {
			return err
		}

		fmt.Println("Updated food vote for:", *enrollment.Food)
	}

	res, err := enrollmentCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return errors.New("no enrollment found to delete")
	}

	return nil
}

// ✅ 4. Admin ดู Student ที่ลงทะเบียนในกิจกรรม พร้อมรายละเอียด
func GetStudentsByActivity(activityID primitive.ObjectID) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 ดึง `activityItemId` ทั้งหมดที่อยู่ภายใต้ `activityId`
	activityItemIDs := []primitive.ObjectID{}
	cursor, err := activityItemCollection.Find(ctx, bson.M{"activityId": activityID})
	if err != nil {
		return nil, fmt.Errorf("error fetching activity items: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var item struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&item); err != nil {
			log.Println("Error decoding activity item:", err)
			continue
		}
		activityItemIDs = append(activityItemIDs, item.ID)
	}

	if len(activityItemIDs) == 0 {
		return []bson.M{}, nil
	}

	// 🔍 ดึงข้อมูลนักศึกษาที่ลงทะเบียนในทุก `activityItemId`
	pipeline := mongo.Pipeline{
		// 1️⃣ Match Enrollment ตาม `activityItemIds`
		bson.D{{Key: "$match", Value: bson.M{"activityItemId": bson.M{"$in": activityItemIDs}}}},

		// 2️⃣ Lookup Student Collection
		bson.D{{
			Key: "$lookup", Value: bson.M{
				"from":         "students",
				"localField":   "studentId",
				"foreignField": "_id",
				"as":           "studentDetails",
			},
		}},
		bson.D{{Key: "$unwind", Value: "$studentDetails"}},

		// 3️⃣ Lookup Major Collection
		bson.D{{
			Key: "$lookup", Value: bson.M{
				"from":         "majors",
				"localField":   "studentDetails.majorId",
				"foreignField": "_id",
				"as":           "majorDetails",
			},
		}},
		bson.D{{Key: "$unwind", Value: bson.M{"path": "$majorDetails", "preserveNullAndEmptyArrays": true}}},

		// 4️⃣ Lookup ActivityItems เพื่อดึง `name`
		bson.D{{
			Key: "$lookup", Value: bson.M{
				"from":         "activityItems",
				"localField":   "activityItemId",
				"foreignField": "_id",
				"as":           "activityItemDetails",
			},
		}},
		bson.D{{Key: "$unwind", Value: "$activityItemDetails"}},

		// 5️⃣ Project ข้อมูลที่ต้องการ
		bson.D{{
			Key: "$project", Value: bson.M{
				"activityItemId":   "$activityItemId",
				"activityItemName": "$activityItemDetails.name", // ✅ เพิ่ม Name ของ ActivityItem
				"student": bson.M{
					"id":        "$studentDetails._id",
					"code":      "$studentDetails.code",
					"name":      "$studentDetails.name",
					"email":     "$studentDetails.email",
					"status":    "$studentDetails.status",
					"major":     "$majorDetails.majorName",
					"softSkill": "$studentDetails.softSkill",
					"hardSkill": "$studentDetails.hardSkill",
				},
			},
		}},

		// 6️⃣ Group นักศึกษาตาม `activityItemId`
		bson.D{{
			Key: "$group", Value: bson.M{
				"_id":      "$activityItemId",
				"id":       bson.M{"$first": "$activityItemId"},
				"name":     bson.M{"$first": "$activityItemName"}, // ✅ เพิ่ม Name
				"students": bson.M{"$push": bson.M{"student": "$student"}},
			},
		}},

		// 7️⃣ Group ตาม `activityId`
		bson.D{{
			Key: "$group", Value: bson.M{
				"_id":            activityID,
				"activityId":     bson.M{"$first": activityID},
				"activityItemId": bson.M{"$push": bson.M{"id": "$id", "name": "$name", "students": "$students"}}, // ✅ เพิ่ม Name ลงใน activityItemId
			},
		}},

		// 8️⃣ Remove `_id`
		bson.D{{Key: "$unset", Value: "_id"}},
	}

	cursor, err = enrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation error: %v", err)
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	if len(result) == 0 {
		return []bson.M{}, nil
	}

	return result, nil
}

// ✅ 5. ดึงข้อมูลเฉพาะ Activity ที่ Student ลงทะเบียนไว้ (1 ตัว)
func GetEnrollmentByStudentAndActivity(studentID, activityItemID primitive.ObjectID) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 ตรวจสอบว่ามี Enrollment หรือไม่
	count, err := enrollmentCollection.CountDocuments(ctx, bson.M{
		"studentId":      studentID,
		"activityItemId": activityItemID,
	})
	if err != nil {
		return nil, fmt.Errorf("database error: %v", err)
	}
	if count == 0 {
		return nil, errors.New("Enrollment not found")
	}

	// 🔄 Aggregate Query เพื่อดึงเฉพาะ Enrollment ที่ตรงกับ Student และ ActivityItem
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"studentId": studentID, "activityItemId": activityItemID}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "activityItems",
			"localField":   "activityItemId",
			"foreignField": "_id",
			"as":           "activityItemDetails",
		}}},
		bson.D{{Key: "$unwind", Value: "$activityItemDetails"}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "activitys",
			"localField":   "activityItemDetails.activityId",
			"foreignField": "_id",
			"as":           "activityDetails",
		}}},
		bson.D{{Key: "$unwind", Value: "$activityDetails"}},
		bson.D{{Key: "$project", Value: bson.M{
			"_id":              0,
			"id":               "$_id",
			"registrationDate": "$registrationDate",
			"studentId":        "$studentId",
			"activity": bson.M{
				"id":              "$activityDetails._id",
				"name":            "$activityDetails.name",
				"type":            "$activityDetails.type",
				"adminId":         "$activityDetails.adminId",
				"activityStateId": "$activityDetails.activityStateId",
				"skillId":         "$activityDetails.skillId",
				"majorIds":        "$activityDetails.majorIds",
				"activityItems": bson.M{
					"id":              "$activityItemDetails._id",
					"activityId":      "$activityItemDetails.activityId",
					"name":            "$activityItemDetails.name",
					"maxParticipants": "$activityItemDetails.maxParticipants",
					"description":     "$activityItemDetails.description",
					"room":            "$activityItemDetails.room",
					"startDate":       "$activityItemDetails.startDate",
					"endDate":         "$activityItemDetails.endDate",
					"duration":        "$activityItemDetails.duration",
					"operator":        "$activityItemDetails.operator",
					"hour":            "$activityItemDetails.hour",
				},
			},
		}}},
	}

	cursor, err := enrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation error: %v", err)
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	// ถ้าไม่มีข้อมูล ให้ส่ง `nil`
	if len(result) == 0 {
		return nil, errors.New("Enrollment not found")
	}

	return result[0], nil // ✅ ส่ง Object เดียว
}
func IsStudentEnrolledInActivity(studentID, activityID primitive.ObjectID) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1️⃣ ดึง activityItems ทั้งหมดที่อยู่ใน activity นี้
	cursor, err := activityItemCollection.Find(ctx, bson.M{"activityId": activityID})
	if err != nil {
		return false, err
	}
	defer cursor.Close(ctx)

	itemIDs := []primitive.ObjectID{}
	for cursor.Next(ctx) {
		var item struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&item); err == nil {
			itemIDs = append(itemIDs, item.ID)
		}
	}

	if len(itemIDs) == 0 {
		return false, nil // ไม่มีกิจกรรมย่อยเลย
	}

	// 2️⃣ ตรวจสอบว่านิสิตลงทะเบียนใน item ใดๆ เหล่านี้หรือไม่
	filter := bson.M{
		"studentId":      studentID,
		"activityItemId": bson.M{"$in": itemIDs},
	}

	count, err := enrollmentCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func isTimeOverlap(start1, end1, start2, end2 string) bool {
	// ตัวอย่าง: 09:00 < 10:00 -> true (มีเวลาทับซ้อน)
	return !(end1 <= start2 || end2 <= start1)
}
