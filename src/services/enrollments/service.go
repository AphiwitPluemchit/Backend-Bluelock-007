package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/activities"
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ✅ 1. Student ลงทะเบียนกิจกรรม (ลงซ้ำไม่ได้ + เช็ค major + กันเวลาทับซ้อน)
func RegisterStudent(activityItemID, studentID primitive.ObjectID, food *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1) ตรวจว่า ActivityItem มีจริงไหม
	var activityItem models.ActivityItem
	if err := DB.ActivityItemCollection.FindOne(ctx, bson.M{"_id": activityItemID}).Decode(&activityItem); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("activity item not found")
		}
		return err
	}

	// 2) ถ้ามีการเลือกอาหาร: +1 vote ให้ foodName ที่ตรงกันใน Activity
	if food != nil {
		activityID := activityItem.ActivityID

		filter := bson.M{"_id": activityID}
		update := bson.M{
			"$inc": bson.M{"foodVotes.$[elem].vote": 1},
		}
		arrayFilter := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []any{
				bson.M{"elem.foodName": *food},
			},
		})

		if _, err := DB.ActivityCollection.UpdateOne(ctx, filter, update, arrayFilter); err != nil {
			return fmt.Errorf("update food vote failed: %w", err)
		}
		// fmt.Println("Updated food vote for:", *food)
	}

	// 3) กันเวลาทับซ้อนกับ enrollment ที่เคยลงไว้แล้ว
	existingEnrollmentsCursor, err := DB.EnrollmentCollection.Find(ctx, bson.M{"studentId": studentID})
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
		if err := DB.ActivityItemCollection.FindOne(ctx, bson.M{"_id": existing.ActivityItemID}).Decode(&existingItem); err != nil {
			continue
		}

		// เปรียบเทียบวันเวลา
		for _, dOld := range existingItem.Dates {
			for _, dNew := range activityItem.Dates {
				if dOld.Date == dNew.Date { // วันเดียวกัน
					if isTimeOverlap(dOld.Stime, dOld.Etime, dNew.Stime, dNew.Etime) {
						return errors.New("ไม่สามารถลงทะเบียนได้ เนื่องจากมีกิจกรรมที่เวลาเดียวกันอยู่แล้ว")
					}
				}
			}
		}
	}

	// 4) โหลด student และเช็ค major ให้ตรงกับ activityItem.Majors (ถ้ามีจำกัด)
	var student models.Student
	if err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("student not found")
		}
		return err
	}

	// ✅ เช็คสาขา: กิจกรรมอนุญาตเฉพาะบาง major
	if len(activityItem.Majors) > 0 {
		allowed := false
		for _, m := range activityItem.Majors {
			log.Println(activityItem.Majors)
			log.Println(student.Major)
			if strings.EqualFold(m, student.Major) { // ปลอดภัยต่อเคสตัวพิมพ์เล็ก/ใหญ่
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.New("ไม่สามารถลงทะเบียนได้: สาขาไม่ตรงกับเงื่อนไขของกิจกรรม")
		}
	}

	// (ถ้าต้องการเช็คชั้นปีด้วย ให้เพิ่มเงื่อนไขจาก activityItem.StudentYears ที่นี่ได้)

	// 5) กันเต็มโควต้า
	if activityItem.MaxParticipants != nil && activityItem.EnrollmentCount >= *activityItem.MaxParticipants {
		return errors.New("ไม่สามารถลงทะเบียนได้ เนื่องจากจำนวนผู้เข้าร่วมเต็มแล้ว")
	}

	// 6) กันลงซ้ำ
	count, err := DB.EnrollmentCollection.CountDocuments(ctx, bson.M{
		"activityItemId": activityItemID,
		"studentId":      studentID,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("already enrolled in this activity")
	}

	// 7) Insert enrollment
	newEnrollment := models.Enrollment{
		ID:               primitive.NewObjectID(),
		StudentID:        studentID,
		ActivityItemID:   activityItemID,
		RegistrationDate: time.Now(),
		Food:             food,
	}
	if _, err := DB.EnrollmentCollection.InsertOne(ctx, newEnrollment); err != nil {
		return err
	}

	// 8) เพิ่ม enrollmentcount +1 ใน activityItems
	if _, err := DB.ActivityItemCollection.UpdateOne(
		ctx,
		bson.M{"_id": activityItemID},
		bson.M{"$inc": bson.M{"enrollmentcount": 1}},
	); err != nil {
		return fmt.Errorf("เพิ่ม enrollmentcount ไม่สำเร็จ: %w", err)
	}

	return nil
}

// ✅ 2. ดึงกิจกรรมทั้งหมดที่ Student ลงทะเบียนไปแล้ว พร้อม pagination และ filter
func GetEnrollmentsByStudent(studentID primitive.ObjectID, params models.PaginationParams, skillFilter []string) ([]models.ActivityDto, int64, int, error) {
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
	cur, err := DB.EnrollmentCollection.Aggregate(ctx, enrollmentStage)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error fetching enrollments: %v", err)
	}
	var enrollmentResult []bson.M
	if err := cur.All(ctx, &enrollmentResult); err != nil || len(enrollmentResult) == 0 {
		return []models.ActivityDto{}, 0, 0, nil
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

	total, err := DB.ActivityCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	pipeline := activities.GetActivitiesPipeline(filter, params.SortBy, sort[0].Value.(int), skip, int64(params.Limit), []string{}, []int{})
	cursor, err := DB.ActivityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var activities []models.ActivityDto
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
	err := DB.EnrollmentCollection.FindOne(ctx, filter).Decode(&enrollment)
	if err != nil {
		return err
	}

	var activityItem models.ActivityItem
	if err := DB.ActivityItemCollection.FindOne(ctx, bson.M{"_id": enrollment.ActivityItemID}).Decode(&activityItem); err != nil {
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
		_, err := DB.ActivityCollection.UpdateOne(ctx, filter, update, arrayFilter)
		if err != nil {
			return err
		}

		fmt.Println("Updated food vote for:", *enrollment.Food)
	}

	res, err := DB.EnrollmentCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return errors.New("no enrollment found to delete")
	}

	// ✅ ลบ enrollmentcount -1 จาก activityItem
	_, err = DB.ActivityItemCollection.UpdateOne(ctx,
		bson.M{"_id": enrollment.ActivityItemID},
		bson.M{"$inc": bson.M{"enrollmentcount": -1}},
	)
	if err != nil {
		return fmt.Errorf("ลด enrollmentcount ไม่สำเร็จ: %w", err)
	}

	return nil
}

// ✅ 4. Admin ดู Student ที่ลงทะเบียนในกิจกรรม พร้อมรายละเอียด
func GetStudentsByActivity(activityID primitive.ObjectID) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 ดึง `activityItemId` ทั้งหมดที่อยู่ภายใต้ `activityId`
	activityItemIDs := []primitive.ObjectID{}
	cursor, err := DB.ActivityItemCollection.Find(ctx, bson.M{"activityId": activityID})
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

	cursor, err = DB.EnrollmentCollection.Aggregate(ctx, pipeline)
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
	count, err := DB.EnrollmentCollection.CountDocuments(ctx, bson.M{
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
			"from":         "activityIte	ms",
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

	cursor, err := DB.EnrollmentCollection.Aggregate(ctx, pipeline)
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

// ✅ 6. ดึงข้อมูล Enrollment ของ Student ใน Activity (รวม IsStudentEnrolledInActivity + GetEnrollmentByStudentAndActivity)
func GetStudentEnrollmentInActivity(studentID, activityID primitive.ObjectID) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1️⃣ ดึง activityItems ทั้งหมดใน activity นี้
	cursor, err := DB.ActivityItemCollection.Find(ctx, bson.M{"activityId": activityID})
	if err != nil {
		return nil, fmt.Errorf("error fetching activity items: %v", err)
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
		return nil, errors.New("No activity items found for this activity")
	}

	// 2️⃣ ตรวจสอบว่านิสิตลงทะเบียนใน item ใดๆ เหล่านี้หรือไม่
	filter := bson.M{
		"studentId":      studentID,
		"activityItemId": bson.M{"$in": itemIDs},
	}

	var enrollment struct {
		ID             primitive.ObjectID `bson:"_id"`
		ActivityItemID primitive.ObjectID `bson:"activityItemId"`
	}
	err = DB.EnrollmentCollection.FindOne(ctx, filter).Decode(&enrollment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("Student not enrolled in this activity")
		}
		return nil, fmt.Errorf("database error: %v", err)
	}

	// 3️⃣ Aggregate Query เพื่อดึงข้อมูลเต็มของ Enrollment ที่ตรงกับ Student และ ActivityItem
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"studentId": studentID, "activityItemId": enrollment.ActivityItemID}}},
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
			"food":             "$food",
			"activity": bson.M{
				"id":              "$activityDetails._id",
				"name":            "$activityDetails.name",
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
					"rooms":           "$activityItemDetails.rooms",
					"startDate":       "$activityItemDetails.startDate",
					"endDate":         "$activityItemDetails.endDate",
					"duration":        "$activityItemDetails.duration",
					"operator":        "$activityItemDetails.operator",
					"hour":            "$activityItemDetails.hour",
				},
			},
		}}},
	}

	cursor, err = DB.EnrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation error: %v", err)
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	if len(result) == 0 {
		return nil, errors.New("Enrollment not found")
	}

	return result[0], nil
}

func isTimeOverlap(start1, end1, start2, end2 string) bool {
	// ตัวอย่าง: 09:00 < 10:00 -> true (มีเวลาทับซ้อน)
	return !(end1 <= start2 || end2 <= start1)
}

func IsStudentEnrolled(studentId string, activityItemId string) bool {
	sID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(activityItemId)

	if err1 != nil || err2 != nil {
		log.Printf("Invalid ObjectID: studentId=%s, activityItemId=%s", studentId, activityItemId)
		return false
	}

	filter := bson.M{
		"studentId":      sID,
		"activityItemId": aID,
	}

	count, err := DB.EnrollmentCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		log.Printf("MongoDB error when checking enrollment: %v", err)
		return false
	}

	return count > 0
}

// FindEnrolledItem คืน activityItemId ที่นิสิตลงทะเบียนไว้ใน activityId นี้
func FindEnrolledItem(userId string, activityId string) (string, bool) {
	uID, _ := primitive.ObjectIDFromHex(userId)
	aID, _ := primitive.ObjectIDFromHex(activityId)

	// 1. ดึง enrollments ทั้งหมดของ userId
	cursor, err := DB.EnrollmentCollection.Find(context.TODO(), bson.M{
		"studentId": uID, // หรือ "userId" ถ้าคุณใช้ชื่อนี้
	})
	if err != nil {
		return "", false
	}
	defer cursor.Close(context.TODO())

	// 2. เช็กแต่ละรายการว่า activityItemId → activityId ตรงหรือไม่
	for cursor.Next(context.TODO()) {
		var enrollment models.Enrollment
		if err := cursor.Decode(&enrollment); err != nil {
			continue
		}

		var item models.ActivityItem
		err := DB.ActivityItemCollection.FindOne(context.TODO(), bson.M{
			"_id": enrollment.ActivityItemID,
		}).Decode(&item)
		if err == nil && item.ActivityID == aID {
			return enrollment.ActivityItemID.Hex(), true
		}
	}

	return "", false
}

// FindEnrolledItems คืน activityItemIds ทั้งหมดที่นิสิตลงทะเบียนไว้ใน activityId นี้
func FindEnrolledItems(userId string, activityId string) ([]string, bool) {
	uID, _ := primitive.ObjectIDFromHex(userId)
	aID, _ := primitive.ObjectIDFromHex(activityId)

	var enrolledItemIDs []string

	// 1. ดึง enrollments ทั้งหมดของ userId
	cursor, err := DB.EnrollmentCollection.Find(context.TODO(), bson.M{
		"studentId": uID, // หรือ "userId" ถ้าคุณใช้ชื่อนี้
	})
	if err != nil {
		return nil, false
	}
	defer cursor.Close(context.TODO())

	// 2. เช็กแต่ละรายการว่า activityItemId → activityId ตรงหรือไม่
	for cursor.Next(context.TODO()) {
		var enrollment models.Enrollment
		if err := cursor.Decode(&enrollment); err != nil {
			continue
		}

		var item models.ActivityItem
		err := DB.ActivityItemCollection.FindOne(context.TODO(), bson.M{
			"_id": enrollment.ActivityItemID,
		}).Decode(&item)
		if err == nil && item.ActivityID == aID {
			enrolledItemIDs = append(enrolledItemIDs, enrollment.ActivityItemID.Hex())
		}
	}

	if len(enrolledItemIDs) == 0 {
		return nil, false
	}
	return enrolledItemIDs, true
}

// GetEnrollmentActivityDetails คืนข้อมูล Activity ที่คล้ายกับ activity getOne แต่เอาเฉพาะ item ที่นักเรียนลงทะเบียน
func GetEnrollmentActivityDetails(studentID, activityID primitive.ObjectID) (*models.ActivityDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1️⃣ ดึง activityItems ทั้งหมดใน activity นี้
	cursor, err := DB.ActivityItemCollection.Find(ctx, bson.M{"activityId": activityID})
	if err != nil {
		return nil, fmt.Errorf("error fetching activity items: %v", err)
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
		return nil, errors.New("No activity items found for this activity")
	}

	// 2️⃣ ตรวจสอบว่านิสิตลงทะเบียนใน item ใดๆ เหล่านี้หรือไม่
	filter := bson.M{
		"studentId":      studentID,
		"activityItemId": bson.M{"$in": itemIDs},
	}

	var enrollment struct {
		ID             primitive.ObjectID `bson:"_id"`
		ActivityItemID primitive.ObjectID `bson:"activityItemId"`
	}
	err = DB.EnrollmentCollection.FindOne(ctx, filter).Decode(&enrollment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("Student not enrolled in this activity")
		}
		return nil, fmt.Errorf("database error: %v", err)
	}

	// 3️⃣ Aggregate Query เพื่อดึงข้อมูล Activity พร้อม ActivityItems ที่นักเรียนลงทะเบียน
	pipeline := mongo.Pipeline{
		// Match Activity
		bson.D{{Key: "$match", Value: bson.M{"_id": activityID}}},

		// Lookup ActivityItems
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "activityItems",
			"localField":   "_id",
			"foreignField": "activityId",
			"as":           "activityItems",
		}}},

		// Unwind ActivityItems
		bson.D{{Key: "$unwind", Value: "$activityItems"}},

		// Lookup Enrollments เพื่อเช็คว่านักเรียนลงทะเบียนใน item นี้หรือไม่
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "enrollments",
			"localField":   "activityItems._id",
			"foreignField": "activityItemId",
			"as":           "enrollments",
		}}},

		// Match เฉพาะ ActivityItems ที่นักเรียนลงทะเบียน
		bson.D{{Key: "$match", Value: bson.M{
			"enrollments": bson.M{
				"$elemMatch": bson.M{"studentId": studentID},
			},
		}}},

		// Group กลับเป็น Activity พร้อม ActivityItems ที่กรองแล้ว
		bson.D{{Key: "$group", Value: bson.M{
			"_id":           "$_id",
			"name":          bson.M{"$first": "$name"},
			"type":          bson.M{"$first": "$type"},
			"activityState": bson.M{"$first": "$activityState"},
			"skill":         bson.M{"$first": "$skill"},
			"file":          bson.M{"$first": "$file"},
			"foodVotes":     bson.M{"$first": "$foodVotes"},
			"endDateEnroll": bson.M{"$first": "$endDateEnroll"},
			"activityItems": bson.M{"$push": "$activityItems"},
		}}},

		// Project ให้ตรงกับ ActivityDto
		bson.D{{Key: "$project", Value: bson.M{
			"_id":           0,
			"id":            "$_id",
			"name":          "$name",
			"type":          "$type",
			"activityState": "$activityState",
			"skill":         "$skill",
			"file":          "$file",
			"foodVotes":     "$foodVotes",
			"endDateEnroll": "$endDateEnroll",
			"activityItems": "$activityItems",
		}}},
	}

	cursor, err = DB.ActivityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation error: %v", err)
	}
	defer cursor.Close(ctx)

	var result models.ActivityDto
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("cursor error: %v", err)
		}
		return &result, nil
	}

	return nil, errors.New("Activity not found")
}

func GetEnrollmentsHistoryByStudent(studentID primitive.ObjectID, params models.PaginationParams, skillFilter []string) ([]models.ActivityHistory, int64, int, error) {
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
	cur, err := DB.EnrollmentCollection.Aggregate(ctx, enrollmentStage)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error fetching enrollments: %v", err)
	}
	var enrollmentResult []bson.M
	if err := cur.All(ctx, &enrollmentResult); err != nil || len(enrollmentResult) == 0 {
		return []models.ActivityHistory{}, 0, 0, nil
	}
	activityIDs := enrollmentResult[0]["activityIds"].(primitive.A)
	activityItemIDs := enrollmentResult[0]["activityItemIds"].(primitive.A)

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

	total, err := DB.ActivityCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	pipeline := activities.GetActivitiesPipeline(filter, params.SortBy, sort[0].Value.(int), skip, int64(params.Limit), []string{}, []int{})
	// เพิ่มขั้นตอนเพื่อกรอง activityItems ให้เหลือเฉพาะที่นิสิตลงทะเบียน
	pipeline = append(pipeline,
		bson.D{{Key: "$addFields", Value: bson.M{
			"activityItems": bson.M{
				"$filter": bson.M{
					"input": "$activityItems",
					"as":    "it",
					"cond":  bson.M{"$in": []interface{}{"$$it._id", activityItemIDs}},
				},
			},
		}}},
	)

	cursor, err := DB.ActivityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var activitiesOut []models.ActivityHistory
	if err := cursor.All(ctx, &activitiesOut); err != nil {
		return nil, 0, 0, err
	}

	// ✅ เติม CheckinoutRecord โดยเรียกใช้ services.GetCheckinStatus (เวลาคืนเป็นโซนไทยอยู่แล้ว)
	for i := range activitiesOut {
		for j := range activitiesOut[i].ActivityItems {
			item := &activitiesOut[i].ActivityItems[j]
			status, _ := GetCheckinStatus(studentID.Hex(), item.ID.Hex())
			if len(status) > 0 {
				item.CheckinoutRecord = status
			}
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return activitiesOut, total, totalPages, nil
}
func GetEnrollmentId(studentID, activityItemID primitive.ObjectID) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var res struct {
		ID primitive.ObjectID `bson:"_id"`
	}

	err := DB.EnrollmentCollection.FindOne(
		ctx,
		bson.M{
			"studentId":      studentID,
			"activityItemId": activityItemID,
		},
		options.FindOne().
			SetProjection(bson.M{"_id": 1}).
			SetSort(bson.D{{Key: "registrationDate", Value: -1}}), // เผื่อมีซ้ำ (ปกติห้ามซ้ำ)
	).Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return primitive.NilObjectID, errors.New("enrollment not found")
		}
		return primitive.NilObjectID, err
	}

	return res.ID, nil
}

type BulkEnrollItem struct {
	StudentCode string  `json:"studentCode"`
	Food        *string `json:"food"`
}

type BulkEnrollResult struct {
	ActivityItemID string                  `json:"activityItemId"`
	TotalRequested int                     `json:"totalRequested"`
	Success        []BulkEnrollSuccessItem `json:"success"`
	Failed         []BulkEnrollFailedItem  `json:"failed"`
}

type BulkEnrollSuccessItem struct {
	StudentCode string `json:"studentCode"`
	StudentID   string `json:"studentId"`
	Message     string `json:"message"`
}

type BulkEnrollFailedItem struct {
	StudentCode string `json:"studentCode"`
	Reason      string `json:"reason"`
}

// ✅ Bulk โดยยังคงใช้กฎจาก RegisterStudent เดิมทุกอย่าง
func RegisterStudentsByCodes(ctx context.Context, activityItemID primitive.ObjectID, items []BulkEnrollItem) (*BulkEnrollResult, error) {
	res := &BulkEnrollResult{
		ActivityItemID: activityItemID.Hex(),
		TotalRequested: len(items),
		Success:        make([]BulkEnrollSuccessItem, 0, len(items)),
		Failed:         make([]BulkEnrollFailedItem, 0),
	}

	// 1) เตรียมรหัสที่ normalize และ dedupe (กันส่งซ้ำ)
	codeSet := make(map[string]struct{}, len(items))
	codes := make([]string, 0, len(items))
	for _, it := range items {
		code := strings.TrimSpace(it.StudentCode)
		if code == "" {
			continue
		}
		if _, ok := codeSet[code]; !ok {
			codeSet[code] = struct{}{}
			codes = append(codes, code)
		}
	}
	// ทำให้มีลำดับคงที่ (optional)
	sort.Strings(codes)

	// 2) ดึง student เป็น batch
	cur, err := DB.StudentCollection.Find(ctx, bson.M{"code": bson.M{"$in": codes}})
	if err != nil {
		return res, fmt.Errorf("failed to query students by codes: %w", err)
	}
	defer cur.Close(ctx)

	codeToStudent := make(map[string]models.Student, len(codes))
	for cur.Next(ctx) {
		var s models.Student
		if derr := cur.Decode(&s); derr == nil {
			codeToStudent[strings.TrimSpace(s.Code)] = s
		}
	}
	if err := cur.Err(); err != nil {
		return res, fmt.Errorf("failed to iterate student cursor: %w", err)
	}

	// 3) วนตาม order ที่ client ส่งมา (report ชัดเจน)
	for _, it := range items {
		code := strings.TrimSpace(it.StudentCode)
		if code == "" {
			res.Failed = append(res.Failed, BulkEnrollFailedItem{
				StudentCode: code,
				Reason:      "studentCode is empty",
			})
			continue
		}

		stu, ok := codeToStudent[code]
		if !ok {
			res.Failed = append(res.Failed, BulkEnrollFailedItem{
				StudentCode: code,
				Reason:      "student not found",
			})
			continue
		}

		// เรียก service เดิมให้ตรวจทุกกฎ (กันชนเวลา/สาขา/เต็มโควต้า/ลงซ้ำ/เพิ่ม foodVotes/เพิ่ม enrollmentcount)
		if err := RegisterStudent(activityItemID, stu.ID, it.Food); err != nil {
			res.Failed = append(res.Failed, BulkEnrollFailedItem{
				StudentCode: code,
				Reason:      err.Error(),
			})
			continue
		}

		res.Success = append(res.Success, BulkEnrollSuccessItem{
			StudentCode: code,
			StudentID:   stu.ID.Hex(),
			Message:     "enrolled",
		})
	}

	return res, nil
}
