package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/programs"
	"Backend-Bluelock-007/src/services/summary_reports"
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
func RegisterStudent(programItemID, studentID primitive.ObjectID, food *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1) ตรวจว่า ProgramItem มีจริงไหม
	var programItem models.ProgramItem
	if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": programItemID}).Decode(&programItem); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("program item not found")
		}
		return err
	}

	// 2) ถ้ามีการเลือกอาหาร: +1 vote ให้ foodName ที่ตรงกันใน Program
	if food != nil {
		programID := programItem.ProgramID

		filter := bson.M{"_id": programID}
		update := bson.M{
			"$inc": bson.M{"foodVotes.$[elem].vote": 1},
		}
		arrayFilter := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []any{
				bson.M{"elem.foodName": *food},
			},
		})

		if _, err := DB.ProgramCollection.UpdateOne(ctx, filter, update, arrayFilter); err != nil {
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

		// ดึง programItem เดิมที่เคยลง
		var existingItem models.ProgramItem
		if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": existing.ProgramItemID}).Decode(&existingItem); err != nil {
			continue
		}

		// เปรียบเทียบวันเวลา
		for _, dOld := range existingItem.Dates {
			for _, dNew := range programItem.Dates {
				if dOld.Date == dNew.Date { // วันเดียวกัน
					if isTimeOverlap(dOld.Stime, dOld.Etime, dNew.Stime, dNew.Etime) {
						return errors.New("ไม่สามารถลงทะเบียนได้ เนื่องจากมีกิจกรรมที่เวลาเดียวกันอยู่แล้ว")
					}
				}
			}
		}
	}

	// 4) โหลด student และเช็ค major ให้ตรงกับ programItem.Majors (ถ้ามีจำกัด)
	var student models.Student
	if err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("student not found")
		}
		return err
	}

	// ✅ เช็คสาขา: กิจกรรมอนุญาตเฉพาะบาง major
	if len(programItem.Majors) > 0 {
		allowed := false
		for _, m := range programItem.Majors {
			log.Println(programItem.Majors)
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

	// (ถ้าต้องการเช็คชั้นปีด้วย ให้เพิ่มเงื่อนไขจาก programItem.StudentYears ที่นี่ได้)

	// 5) กันเต็มโควต้า
	if programItem.MaxParticipants != nil && programItem.EnrollmentCount >= *programItem.MaxParticipants {
		return errors.New("ไม่สามารถลงทะเบียนได้ เนื่องจากจำนวนผู้เข้าร่วมเต็มแล้ว")
	}

	// 6) กันลงซ้ำ
	count, err := DB.EnrollmentCollection.CountDocuments(ctx, bson.M{
		"programItemId": programItemID,
		"studentId":     studentID,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("already enrolled in this program")
	}

	// 7) Insert enrollment
	newEnrollment := models.Enrollment{
		ID:               primitive.NewObjectID(),
		StudentID:        studentID,
		ProgramItemID:    programItemID,
		RegistrationDate: time.Now(),
		Food:             food,
	}
	if _, err := DB.EnrollmentCollection.InsertOne(ctx, newEnrollment); err != nil {
		return err
	}

	// 8) เพิ่ม enrollmentcount +1 ใน programItems
	if _, err := DB.ProgramItemCollection.UpdateOne(
		ctx,
		bson.M{"_id": programItemID},
		bson.M{"$inc": bson.M{"enrollmentcount": 1}},
	); err != nil {
		return fmt.Errorf("เพิ่ม enrollmentcount ไม่สำเร็จ: %w", err)
	}

	// 9) ✅ อัปเดต Summary Report - เพิ่ม Registered count สำหรับแต่ละ date ของ programItem
	for _, date := range programItem.Dates {
		err = summary_reports.UpdateRegisteredCount(programItemID, date.Date, 1)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to update summary report registered count for date %s: %v", date.Date, err)
			// Don't return error here, just log it - we don't want to fail enrollment
			// if summary report update fails
		}
	}

	return nil
}

// ✅ 2. ดึงกิจกรรมทั้งหมดที่ Student ลงทะเบียนไปแล้ว พร้อม pagination และ filter
func GetEnrollmentsByStudent(studentID primitive.ObjectID, params models.PaginationParams, skillFilter []string) ([]models.ProgramDtoWithCheckinoutRecord, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ✅ Step 1: ดึง programItemIds จาก enrollment ที่ student ลงทะเบียน
	matchStage := bson.D{{Key: "$match", Value: bson.M{"studentId": studentID}}}
	lookupProgramItem := bson.D{{Key: "$lookup", Value: bson.M{
		"from":         "Program_Items",
		"localField":   "programItemId",
		"foreignField": "_id",
		"as":           "programItemDetails",
	}}}
	unwindProgramItem := bson.D{{Key: "$unwind", Value: "$programItemDetails"}}
	groupProgramIDs := bson.D{{Key: "$group", Value: bson.M{
		"_id":            nil,
		"programItemIds": bson.M{"$addToSet": "$programItemDetails._id"},
		"programIds":     bson.M{"$addToSet": "$programItemDetails.programId"},
	}}}

	enrollmentStage := mongo.Pipeline{matchStage, lookupProgramItem, unwindProgramItem, groupProgramIDs}
	cur, err := DB.EnrollmentCollection.Aggregate(ctx, enrollmentStage)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error fetching enrollments: %v", err)
	}
	var enrollmentResult []bson.M
	if err := cur.All(ctx, &enrollmentResult); err != nil || len(enrollmentResult) == 0 {
		return []models.ProgramDtoWithCheckinoutRecord{}, 0, 0, nil
	}
	programIDs := enrollmentResult[0]["programIds"].(primitive.A)
	programItemIDs := enrollmentResult[0]["programItemIds"].(primitive.A)

	// ✅ Step 2: Filter + Paginate + Lookup programs เหมือน GetAllPrograms
	skip := int64((params.Page - 1) * params.Limit)
	sort := bson.D{{Key: params.SortBy, Value: 1}}
	if strings.ToLower(params.Order) == "desc" {
		sort[0].Value = -1
	}

	filter := bson.M{"_id": bson.M{"$in": programIDs}}
	if params.Search != "" {
		filter["name"] = bson.M{"$regex": params.Search, "$options": "i"}
	}
	if len(skillFilter) > 0 && skillFilter[0] != "" {
		filter["skill"] = bson.M{"$in": skillFilter}
	}

	total, err := DB.ProgramCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	pipeline := programs.GetProgramsPipeline(filter, params.SortBy, sort[0].Value.(int), skip, int64(params.Limit), []string{}, []int{})
	// กรอง programItems ให้เหลือเฉพาะที่นิสิตลงทะเบียนไว้
	pipeline = append(pipeline,
		bson.D{{Key: "$addFields", Value: bson.M{
			"programItems": bson.M{
				"$filter": bson.M{
					"input": "$programItems",
					"as":    "it",
					"cond":  bson.M{"$in": []interface{}{"$$it._id", programItemIDs}},
				},
			},
		}}},
	)
	cursor, err := DB.ProgramCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var programs []models.ProgramDtoWithCheckinoutRecord
	if err := cursor.All(ctx, &programs); err != nil {
		return nil, 0, 0, err
	}

	// เตรียม latest hour-change-history ต่อ programItem เพื่อตีสถานะ + หา approvedAt
	type histLite struct {
		ProgramItemID primitive.ObjectID  `bson:"programItemId"`
		EnrollmentID  *primitive.ObjectID `bson:"enrollmentId"`
		ChangeType    string              `bson:"changeType"`
		HoursChange   int                 `bson:"hoursChange"`
		ChangedAt     time.Time           `bson:"changedAt"`
	}

	latestByItem := make(map[primitive.ObjectID]histLite)    // ล่าสุดสุด (อะไรก็ได้)
	approvedByItem := make(map[primitive.ObjectID]time.Time) // ล่าสุดที่ถือว่าอนุมัติ

	if len(programItemIDs) > 0 {
		histCur, err := DB.HourChangeHistoryCollection.Find(ctx, bson.M{
			"studentId":     studentID,
			"programItemId": bson.M{"$in": programItemIDs},
			"type":          "program",
		}, options.Find().SetSort(bson.D{{Key: "changedAt", Value: -1}}))
		if err == nil {
			for histCur.Next(ctx) {
				var h histLite
				if derr := histCur.Decode(&h); derr == nil {
					// 1) เก็บตัวล่าสุด (ใช้ตีสถานะ)
					if _, ok := latestByItem[h.ProgramItemID]; !ok {
						latestByItem[h.ProgramItemID] = h
					}
					// 2) เก็บตัวล่าสุดที่ "อนุมัติ"
					if _, ok := approvedByItem[h.ProgramItemID]; !ok {
						if h.ChangeType == "add" || h.ChangeType == "no_change" || (h.HoursChange >= 0 && h.ChangeType == "") {
							approvedByItem[h.ProgramItemID] = h.ChangedAt
						}
					}
				}
			}
			_ = histCur.Close(ctx)
		}
	}

	for i := range programs {
		for j := range programs[i].ProgramItems {
			item := &programs[i].ProgramItems[j]

			// check-in/out times
			statusRecs, _ := GetCheckinStatus(studentID.Hex(), item.ID.Hex())
			if len(statusRecs) > 0 {
				item.CheckinoutRecord = statusRecs
			}

			// default: 1 ยังไม่เข้าร่วม
			st := 1
			if h, ok := latestByItem[item.ID]; ok {
				if h.ChangeType == "remove" || h.HoursChange < 0 {
					st = 3 // ลงทะเบียนแต่ไม่เข้า/ตัดชั่วโมง
				} else if h.ChangeType == "add" || h.ChangeType == "no_change" || (h.HoursChange >= 0 && h.ChangeType == "") {
					st = 2 // เข้าร่วม/อนุมัติแล้ว
				} else {
					st = 1
				}
			}
			item.Status = &st

			// ✅ ใส่วันที่อนุมัติ (ถ้ามีและสถานะเป็น 2)
			if st == 2 {
				if t, ok := approvedByItem[item.ID]; ok {
					tt := t
					item.ApprovedAt = &tt
				}
			}
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return programs, total, totalPages, nil
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

	var programItem models.ProgramItem
	if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": enrollment.ProgramItemID}).Decode(&programItem); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("program item not found")
		}
		return err
	}

	if enrollment.Food != nil {
		programID := programItem.ProgramID

		// ✅ Update -1 vote ของ foodName ที่ตรงกับชื่ออาหาร
		filter := bson.M{"_id": programID}
		update := bson.M{
			"$inc": bson.M{"foodVotes.$[elem].vote": -1},
		}
		arrayFilter := options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []any{
				bson.M{"elem.foodName": *enrollment.Food},
			},
		})

		// ✅ Run update
		_, err := DB.ProgramCollection.UpdateOne(ctx, filter, update, arrayFilter)
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

	// ✅ ลบ enrollmentcount -1 จาก programItem
	_, err = DB.ProgramItemCollection.UpdateOne(ctx,
		bson.M{"_id": enrollment.ProgramItemID},
		bson.M{"$inc": bson.M{"enrollmentcount": -1}},
	)
	if err != nil {
		return fmt.Errorf("ลด enrollmentcount ไม่สำเร็จ: %w", err)
	}

	// ✅ อัปเดต Summary Report - ลด Registered count สำหรับแต่ละ date ของ programItem
	for _, date := range programItem.Dates {
		err = summary_reports.UpdateRegisteredCount(enrollment.ProgramItemID, date.Date, -1)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to update summary report registered count for date %s: %v", date.Date, err)
			// Don't return error here, just log it - we don't want to fail unenrollment
			// if summary report update fails
		}
	}

	return nil
}

// ✅ 4. Admin ดู Student ที่ลงทะเบียนในกิจกรรม พร้อมรายละเอียด
func GetStudentsByProgram(programID primitive.ObjectID) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 ดึง `programItemId` ทั้งหมดที่อยู่ภายใต้ `programId`
	programItemIDs := []primitive.ObjectID{}
	cursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID})
	if err != nil {
		return nil, fmt.Errorf("error fetching program items: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var item struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&item); err != nil {
			log.Println("Error decoding program item:", err)
			continue
		}
		programItemIDs = append(programItemIDs, item.ID)
	}

	if len(programItemIDs) == 0 {
		return []bson.M{}, nil
	}

	// 🔍 ดึงข้อมูลนักศึกษาที่ลงทะเบียนในทุก `programItemId`
	pipeline := mongo.Pipeline{
		// 1️⃣ Match Enrollment ตาม `programItemIds`
		bson.D{{Key: "$match", Value: bson.M{"programItemId": bson.M{"$in": programItemIDs}}}},

		// 2️⃣ Lookup Student Collection
		bson.D{{
			Key: "$lookup", Value: bson.M{
				"from":         "Students",
				"localField":   "studentId",
				"foreignField": "_id",
				"as":           "studentDetails",
			},
		}},
		bson.D{{Key: "$unwind", Value: "$studentDetails"}},

		// 4️⃣ Lookup ProgramItems เพื่อดึง `name`
		bson.D{{
			Key: "$lookup", Value: bson.M{
				"from":         "Program_Items",
				"localField":   "programItemId",
				"foreignField": "_id",
				"as":           "programItemDetails",
			},
		}},
		bson.D{{Key: "$unwind", Value: "$programItemDetails"}},

		// 5️⃣ Project ข้อมูลที่ต้องการ
		bson.D{{
			Key: "$project", Value: bson.M{
				"programItemId":   "$programItemId",
				"programItemName": "$programItemDetails.name", // ✅ เพิ่ม Name ของ ProgramItem
				"student": bson.M{
					"id":        "$studentDetails._id",
					"code":      "$studentDetails.code",
					"name":      "$studentDetails.name",
					"email":     "$studentDetails.email",
					"status":    "$studentDetails.status",
					"major":     "$studentDetails.major",
					"softSkill": "$studentDetails.softSkill",
					"hardSkill": "$studentDetails.hardSkill",
				},
			},
		}},

		// 6️⃣ Group นักศึกษาตาม `programItemId`
		bson.D{{
			Key: "$group", Value: bson.M{
				"_id":      "$programItemId",
				"id":       bson.M{"$first": "$programItemId"},
				"name":     bson.M{"$first": "$programItemName"}, // ✅ เพิ่ม Name
				"students": bson.M{"$push": bson.M{"student": "$student"}},
			},
		}},

		// 7️⃣ Group ตาม `programId`
		bson.D{{
			Key: "$group", Value: bson.M{
				"_id":           programID,
				"programId":     bson.M{"$first": programID},
				"programItemId": bson.M{"$push": bson.M{"id": "$id", "name": "$name", "students": "$students"}}, // ✅ เพิ่ม Name ลงใน programItemId
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

// ✅ 5. ดึงข้อมูลเฉพาะ Program ที่ Student ลงทะเบียนไว้ (1 ตัว)
func GetEnrollmentByStudentAndProgram(studentID, programItemID primitive.ObjectID) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 ตรวจสอบว่ามี Enrollment หรือไม่
	count, err := DB.EnrollmentCollection.CountDocuments(ctx, bson.M{
		"studentId":     studentID,
		"programItemId": programItemID,
	})
	if err != nil {
		return nil, fmt.Errorf("database error: %v", err)
	}
	if count == 0 {
		return nil, errors.New("Enrollment not found")
	}

	// 🔄 Aggregate Query เพื่อดึงเฉพาะ Enrollment ที่ตรงกับ Student และ ProgramItem
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"studentId": studentID, "programItemId": programItemID}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "Program_Items",
			"localField":   "programItemId",
			"foreignField": "_id",
			"as":           "programItemDetails",
		}}},
		bson.D{{Key: "$unwind", Value: "$programItemDetails"}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "Programs",
			"localField":   "programItemDetails.programId",
			"foreignField": "_id",
			"as":           "programDetails",
		}}},
		bson.D{{Key: "$unwind", Value: "$programDetails"}},
		bson.D{{Key: "$project", Value: bson.M{
			"_id":              0,
			"id":               "$_id",
			"registrationDate": "$registrationDate",
			"studentId":        "$studentId",
			"program": bson.M{
				"id":             "$programDetails._id",
				"name":           "$programDetails.name",
				"type":           "$programDetails.type",
				"adminId":        "$programDetails.adminId",
				"programStateId": "$programDetails.programStateId",
				"skillId":        "$programDetails.skillId",
				"majorIds":       "$programDetails.majorIds",
				"programItems": bson.M{
					"id":              "$programItemDetails._id",
					"programId":       "$programItemDetails.programId",
					"name":            "$programItemDetails.name",
					"maxParticipants": "$programItemDetails.maxParticipants",
					"description":     "$programItemDetails.description",
					"room":            "$programItemDetails.room",
					"startDate":       "$programItemDetails.startDate",
					"endDate":         "$programItemDetails.endDate",
					"duration":        "$programItemDetails.duration",
					"operator":        "$programItemDetails.operator",
					"hour":            "$programItemDetails.hour",
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

// ✅ 6. ดึงข้อมูล Enrollment ของ Student ใน Program (รวม IsStudentEnrolledInProgram + GetEnrollmentByStudentAndProgram)
func GetStudentEnrollmentInProgram(studentID, programID primitive.ObjectID) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1️⃣ ดึง programItems ทั้งหมดใน program นี้
	cursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID})
	if err != nil {
		return nil, fmt.Errorf("error fetching program items: %v", err)
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
		return nil, errors.New("No program items found for this program")
	}

	// 2️⃣ ตรวจสอบว่านิสิตลงทะเบียนใน item ใดๆ เหล่านี้หรือไม่
	filter := bson.M{
		"studentId":     studentID,
		"programItemId": bson.M{"$in": itemIDs},
	}

	var enrollment struct {
		ID            primitive.ObjectID `bson:"_id"`
		ProgramItemID primitive.ObjectID `bson:"programItemId"`
	}
	err = DB.EnrollmentCollection.FindOne(ctx, filter).Decode(&enrollment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("Student not enrolled in this program")
		}
		return nil, fmt.Errorf("database error: %v", err)
	}

	// 3️⃣ Aggregate Query เพื่อดึงข้อมูลเต็มของ Enrollment ที่ตรงกับ Student และ ProgramItem
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"studentId": studentID, "programItemId": enrollment.ProgramItemID}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "Program_Items",
			"localField":   "programItemId",
			"foreignField": "_id",
			"as":           "programItemDetails",
		}}},
		bson.D{{Key: "$unwind", Value: "$programItemDetails"}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "Programs",
			"localField":   "programItemDetails.programId",
			"foreignField": "_id",
			"as":           "programDetails",
		}}},
		bson.D{{Key: "$unwind", Value: "$programDetails"}},
		bson.D{{Key: "$project", Value: bson.M{
			"_id":              0,
			"id":               "$_id",
			"registrationDate": "$registrationDate",
			"studentId":        "$studentId",
			"food":             "$food",
			"program": bson.M{
				"id":             "$programDetails._id",
				"name":           "$programDetails.name",
				"adminId":        "$programDetails.adminId",
				"programStateId": "$programDetails.programStateId",
				"skillId":        "$programDetails.skillId",
				"majorIds":       "$programDetails.majorIds",
				"programItems": bson.M{
					"id":              "$programItemDetails._id",
					"programId":       "$programItemDetails.programId",
					"name":            "$programItemDetails.name",
					"maxParticipants": "$programItemDetails.maxParticipants",
					"description":     "$programItemDetails.description",
					"rooms":           "$programItemDetails.rooms",
					"startDate":       "$programItemDetails.startDate",
					"endDate":         "$programItemDetails.endDate",
					"duration":        "$programItemDetails.duration",
					"operator":        "$programItemDetails.operator",
					"hour":            "$programItemDetails.hour",
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

func IsStudentEnrolled(studentId string, programItemId string) bool {
	sID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(programItemId)

	if err1 != nil || err2 != nil {
		log.Printf("Invalid ObjectID: studentId=%s, programItemId=%s", studentId, programItemId)
		return false
	}

	filter := bson.M{
		"studentId":     sID,
		"programItemId": aID,
	}

	count, err := DB.EnrollmentCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		log.Printf("MongoDB error when checking enrollment: %v", err)
		return false
	}

	return count > 0
}

// FindEnrolledItem คืน programItemId ที่นิสิตลงทะเบียนไว้ใน programId นี้
func FindEnrolledItem(userId string, programId string) (string, bool) {
	uID, _ := primitive.ObjectIDFromHex(userId)
	aID, _ := primitive.ObjectIDFromHex(programId)

	// 1. ดึง enrollments ทั้งหมดของ userId
	cursor, err := DB.EnrollmentCollection.Find(context.TODO(), bson.M{
		"studentId": uID, // หรือ "userId" ถ้าคุณใช้ชื่อนี้
	})
	if err != nil {
		return "", false
	}
	defer cursor.Close(context.TODO())

	// 2. เช็กแต่ละรายการว่า programItemId → programId ตรงหรือไม่
	for cursor.Next(context.TODO()) {
		var enrollment models.Enrollment
		if err := cursor.Decode(&enrollment); err != nil {
			continue
		}

		var item models.ProgramItem
		err := DB.ProgramItemCollection.FindOne(context.TODO(), bson.M{
			"_id": enrollment.ProgramItemID,
		}).Decode(&item)
		if err == nil && item.ProgramID == aID {
			return enrollment.ProgramItemID.Hex(), true
		}
	}

	return "", false
}

// FindEnrolledItems คืน programItemIds ทั้งหมดที่นิสิตลงทะเบียนไว้ใน programId นี้
func FindEnrolledItems(userId string, programId string) ([]string, bool) {
	uID, _ := primitive.ObjectIDFromHex(userId)
	aID, _ := primitive.ObjectIDFromHex(programId)

	var enrolledItemIDs []string

	// 1. ดึง enrollments ทั้งหมดของ userId
	cursor, err := DB.EnrollmentCollection.Find(context.TODO(), bson.M{
		"studentId": uID, // หรือ "userId" ถ้าคุณใช้ชื่อนี้
	})
	if err != nil {
		return nil, false
	}
	defer cursor.Close(context.TODO())

	// 2. เช็กแต่ละรายการว่า programItemId → programId ตรงหรือไม่
	for cursor.Next(context.TODO()) {
		var enrollment models.Enrollment
		if err := cursor.Decode(&enrollment); err != nil {
			continue
		}

		var item models.ProgramItem
		err := DB.ProgramItemCollection.FindOne(context.TODO(), bson.M{
			"_id": enrollment.ProgramItemID,
		}).Decode(&item)
		if err == nil && item.ProgramID == aID {
			enrolledItemIDs = append(enrolledItemIDs, enrollment.ProgramItemID.Hex())
		}
	}

	if len(enrolledItemIDs) == 0 {
		return nil, false
	}
	return enrolledItemIDs, true
}

// GetEnrollmentProgramDetails คืนข้อมูล Program ที่คล้ายกับ program getOne แต่เอาเฉพาะ item ที่นักเรียนลงทะเบียน
func GetEnrollmentProgramDetails(studentID, programID primitive.ObjectID) (*models.ProgramDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 0️⃣ ตรวจสอบว่า program มีอยู่จริงหรือไม่
	var programExists struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programID}).Decode(&programExists)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("Program not found")
		}
		return nil, fmt.Errorf("error checking program existence: %v", err)
	}

	// 1️⃣ ดึง programItems ทั้งหมดใน program นี้
	cursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID})
	if err != nil {
		return nil, fmt.Errorf("error fetching program items: %v", err)
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

	// Debug: Log the program items found
	fmt.Printf("DEBUG: Found %d program items for program %s: %v\n", len(itemIDs), programID.Hex(), itemIDs)

	if len(itemIDs) == 0 {
		return nil, errors.New("No program items found for this program")
	}

	// 2️⃣ ตรวจสอบว่านิสิตลงทะเบียนใน item ใดๆ เหล่านี้หรือไม่
	filter := bson.M{
		"studentId":     studentID,
		"programItemId": bson.M{"$in": itemIDs},
	}

	// Debug: Log the enrollment filter
	fmt.Printf("DEBUG: Checking enrollment with filter: %+v\n", filter)

	var enrollment struct {
		ID            primitive.ObjectID `bson:"_id"`
		ProgramItemID primitive.ObjectID `bson:"programItemId"`
	}
	err = DB.EnrollmentCollection.FindOne(ctx, filter).Decode(&enrollment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Debug: Check if student has any enrollments at all
			var anyEnrollment struct {
				ID primitive.ObjectID `bson:"_id"`
			}
			anyErr := DB.EnrollmentCollection.FindOne(ctx, bson.M{"studentId": studentID}).Decode(&anyEnrollment)
			if anyErr == nil {
				fmt.Printf("DEBUG: Student has enrollments but not in this program\n")
			} else {
				fmt.Printf("DEBUG: Student has no enrollments at all\n")
			}
			return nil, errors.New("Student not enrolled in this program")
		}
		return nil, fmt.Errorf("database error: %v", err)
	}

	fmt.Printf("DEBUG: Found enrollment: %s for programItem: %s\n", enrollment.ID.Hex(), enrollment.ProgramItemID.Hex())

	// 3️⃣ Aggregate Query เพื่อดึงข้อมูล Program พร้อม ProgramItems ที่นักเรียนลงทะเบียน
	pipeline := mongo.Pipeline{
		// Match Program
		bson.D{{Key: "$match", Value: bson.M{"_id": programID}}},

		// Lookup ProgramItems
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "Program_Items",
			"localField":   "_id",
			"foreignField": "programId",
			"as":           "programItems",
		}}},

		// Unwind ProgramItems
		bson.D{{Key: "$unwind", Value: "$programItems"}},

		// Lookup Enrollments เพื่อเช็คว่านักเรียนลงทะเบียนใน item นี้หรือไม่
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "Enrollments",
			"localField":   "programItems._id",
			"foreignField": "programItemId",
			"as":           "enrollments",
		}}},

		// Match เฉพาะ ProgramItems ที่นักเรียนลงทะเบียน
		bson.D{{Key: "$match", Value: bson.M{
			"enrollments": bson.M{
				"$elemMatch": bson.M{"studentId": studentID},
			},
		}}},

		// Group กลับเป็น Program พร้อม ProgramItems ที่กรองแล้ว
		bson.D{{Key: "$group", Value: bson.M{
			"_id":           "$_id",
			"name":          bson.M{"$first": "$name"},
			"type":          bson.M{"$first": "$type"},
			"programState":  bson.M{"$first": "$programState"},
			"skill":         bson.M{"$first": "$skill"},
			"file":          bson.M{"$first": "$file"},
			"foodVotes":     bson.M{"$first": "$foodVotes"},
			"endDateEnroll": bson.M{"$first": "$endDateEnroll"},
			"programItems":  bson.M{"$push": "$programItems"},
		}}},

		// Project ให้ตรงกับ ProgramDto
		bson.D{{Key: "$project", Value: bson.M{
			"_id":           0,
			"id":            "$_id",
			"name":          "$name",
			"type":          "$type",
			"programState":  "$programState",
			"skill":         "$skill",
			"file":          "$file",
			"foodVotes":     "$foodVotes",
			"endDateEnroll": "$endDateEnroll",
			"programItems":  "$programItems",
		}}},
	}

	log.Println(pipeline)
	cursor, err = DB.ProgramCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation error: %v", err)
	}
	defer cursor.Close(ctx)

	var result models.ProgramDto
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("cursor error: %v", err)
		}
		return &result, nil
	}
	log.Println(result)
	return nil, errors.New("Student not enrolled in this program")
}

func GetRegistrationHistoryStatus(studentID primitive.ObjectID, params models.PaginationParams, skillFilter []string) ([]models.ProgramHistory, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ✅ Step 1: ดึง programItemIds จาก enrollment ที่ student ลงทะเบียน
	matchStage := bson.D{{Key: "$match", Value: bson.M{"studentId": studentID}}}
	lookupProgramItem := bson.D{{Key: "$lookup", Value: bson.M{
		"from":         "Program_Items",
		"localField":   "programItemId",
		"foreignField": "_id",
		"as":           "programItemDetails",
	}}}
	unwindProgramItem := bson.D{{Key: "$unwind", Value: "$programItemDetails"}}
	groupProgramIDs := bson.D{{Key: "$group", Value: bson.M{
		"_id":            nil,
		"programItemIds": bson.M{"$addToSet": "$programItemDetails._id"},
		"programIds":     bson.M{"$addToSet": "$programItemDetails.programId"},
	}}}

	enrollmentStage := mongo.Pipeline{matchStage, lookupProgramItem, unwindProgramItem, groupProgramIDs}
	cur, err := DB.EnrollmentCollection.Aggregate(ctx, enrollmentStage)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error fetching enrollments: %v", err)
	}
	var enrollmentResult []bson.M
	if err := cur.All(ctx, &enrollmentResult); err != nil || len(enrollmentResult) == 0 {
		return []models.ProgramHistory{}, 0, 0, nil
	}
	programIDs := enrollmentResult[0]["programIds"].(primitive.A)
	programItemIDs := enrollmentResult[0]["programItemIds"].(primitive.A)

	// ✅ Step 2: Filter + Paginate + Lookup programs เหมือน GetAllPrograms
	skip := int64((params.Page - 1) * params.Limit)
	sort := bson.D{{Key: params.SortBy, Value: 1}}
	if strings.ToLower(params.Order) == "desc" {
		sort[0].Value = -1
	}

	filter := bson.M{"_id": bson.M{"$in": programIDs}}
	if params.Search != "" {
		filter["name"] = bson.M{"$regex": params.Search, "$options": "i"}
	}
	if len(skillFilter) > 0 && skillFilter[0] != "" {
		filter["skill"] = bson.M{"$in": skillFilter}
	}

	total, err := DB.ProgramCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	pipeline := programs.GetProgramsPipeline(filter, params.SortBy, sort[0].Value.(int), skip, int64(params.Limit), []string{}, []int{})
	// เพิ่มขั้นตอนเพื่อกรอง programItems ให้เหลือเฉพาะที่นิสิตลงทะเบียน
	pipeline = append(pipeline,
		bson.D{{Key: "$addFields", Value: bson.M{
			"programItems": bson.M{
				"$filter": bson.M{
					"input": "$programItems",
					"as":    "it",
					"cond":  bson.M{"$in": []interface{}{"$$it._id", programItemIDs}},
				},
			},
		}}},
	)

	cursor, err := DB.ProgramCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var programsOut []models.ProgramHistory
	if err := cursor.All(ctx, &programsOut); err != nil {
		return nil, 0, 0, err
	}

	// ✅ เติม CheckinoutRecord โดยเรียกใช้ services.GetCheckinStatus (เวลาคืนเป็นโซนไทยอยู่แล้ว)
	for i := range programsOut {
		for j := range programsOut[i].ProgramItems {
			item := &programsOut[i].ProgramItems[j]
			status, _ := GetCheckinStatus(studentID.Hex(), item.ID.Hex())
			if len(status) > 0 {
				item.CheckinoutRecord = status
			}
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return programsOut, total, totalPages, nil
}

// RegistrationHistoryStatus กลุ่มประวัติการลงทะเบียนตามสถานะ
// - pending: ยังไม่เข้าร่วม (ไม่มีบันทึกใน Hour_Change_Histories)
// - participated: เข้าร่วมแล้ว (changeType = add หรือ no_change)
// - absent: ลงทะเบียนแต่ไม่ได้เข้าร่วม (changeType = remove)
type EnrollmentsHistoryStatus struct {
	Pending      []map[string]interface{} `json:"pending"`
	Participated []map[string]interface{} `json:"participated"`
	Absent       []map[string]interface{} `json:"absent"`
}

// GetRegistrationHistoryStatus ดึงประวัติการลงทะเบียนของนิสิต แบ่งสถานะจาก Hour_Change_Histories
func GetEnrollmentsHistoryByStudent(studentID primitive.ObjectID) (*EnrollmentsHistoryStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1) ดึง enrollments ของนักศึกษา
	cur, err := DB.EnrollmentCollection.Find(ctx, bson.M{"studentId": studentID})
	if err != nil {
		return nil, fmt.Errorf("error fetching enrollments: %v", err)
	}
	defer cur.Close(ctx)

	type enrollmentLite struct {
		ID               primitive.ObjectID         `bson:"_id"`
		ProgramItemID    primitive.ObjectID         `bson:"programItemId"`
		RegistrationDate time.Time                  `bson:"registrationDate"`
		Food             *string                    `bson:"food"`
		CheckinoutRecord *[]models.CheckinoutRecord `bson:"checkinoutRecord"`
	}

	enrollments := make([]enrollmentLite, 0)
	programItemIDs := make([]primitive.ObjectID, 0)
	for cur.Next(ctx) {
		var e enrollmentLite
		if err := cur.Decode(&e); err == nil {
			enrollments = append(enrollments, e)
			programItemIDs = append(programItemIDs, e.ProgramItemID)
		}
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}
	if len(enrollments) == 0 {
		return &EnrollmentsHistoryStatus{Pending: []map[string]interface{}{}, Participated: []map[string]interface{}{}, Absent: []map[string]interface{}{}}, nil
	}

	// 2) โหลด ProgramItems ที่เกี่ยวข้อง
	itemCur, err := DB.ProgramItemCollection.Find(ctx, bson.M{"_id": bson.M{"$in": programItemIDs}})
	if err != nil {
		return nil, fmt.Errorf("error fetching program items: %v", err)
	}
	defer itemCur.Close(ctx)

	itemMap := make(map[primitive.ObjectID]models.ProgramItem, len(programItemIDs))
	programIDs := make([]primitive.ObjectID, 0)
	for itemCur.Next(ctx) {
		var it models.ProgramItem
		if err := itemCur.Decode(&it); err == nil {
			itemMap[it.ID] = it
			programIDs = append(programIDs, it.ProgramID)
		}
	}
	if err := itemCur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	// 3) โหลด Programs สำหรับเติมชื่อ
	progCur, err := DB.ProgramCollection.Find(ctx, bson.M{"_id": bson.M{"$in": programIDs}})
	if err != nil {
		return nil, fmt.Errorf("error fetching programs: %v", err)
	}
	defer progCur.Close(ctx)

	progMap := make(map[primitive.ObjectID]models.Program, len(programIDs))
	for progCur.Next(ctx) {
		var p models.Program
		if err := progCur.Decode(&p); err == nil {
			progMap[p.ID] = p
		}
	}
	if err := progCur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	// 4) โหลด Hour_Change_Histories ของนิสิตสำหรับ program items เหล่านี้ทั้งหมด
	histCur, err := DB.HourChangeHistoryCollection.Find(ctx, bson.M{
		"studentId":     studentID,
		"programItemId": bson.M{"$in": programItemIDs},
		"type":          "program",
	}, options.Find().SetSort(bson.D{{Key: "changedAt", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("error fetching hour change histories: %v", err)
	}
	defer histCur.Close(ctx)

	type histLite struct {
		ProgramItemID primitive.ObjectID  `bson:"programItemId"`
		EnrollmentID  *primitive.ObjectID `bson:"enrollmentId"`
		ChangeType    string              `bson:"changeType"`
		HoursChange   int                 `bson:"hoursChange"`
		ChangedAt     time.Time           `bson:"changedAt"`
	}

	// เก็บเฉพาะ record ล่าสุดต่อ programItemId
	latestByItem := make(map[primitive.ObjectID]histLite)
	for histCur.Next(ctx) {
		var h histLite
		if err := histCur.Decode(&h); err == nil {
			if _, ok := latestByItem[h.ProgramItemID]; !ok {
				latestByItem[h.ProgramItemID] = h
			}
		}
	}
	if err := histCur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	// 5) จัดหมวดหมู่ผลลัพธ์
	res := &EnrollmentsHistoryStatus{
		Pending:      make([]map[string]interface{}, 0),
		Participated: make([]map[string]interface{}, 0),
		Absent:       make([]map[string]interface{}, 0),
	}

	for _, e := range enrollments {
		it := itemMap[e.ProgramItemID]
		prog := progMap[it.ProgramID]

		// build nested objects with full details
		checkRecs := []models.CheckinoutRecord{}
		if e.CheckinoutRecord != nil {
			checkRecs = append(checkRecs, (*e.CheckinoutRecord)...)
		}
		programItemObj := map[string]interface{}{
			"id":               it.ID.Hex(),
			"programId":        it.ProgramID.Hex(),
			"name":             deref(it.Name),
			"description":      it.Description,
			"studentYears":     it.StudentYears,
			"maxParticipants":  it.MaxParticipants,
			"majors":           it.Majors,
			"rooms":            it.Rooms,
			"operator":         it.Operator,
			"dates":            it.Dates,
			"hour":             it.Hour,
			"enrollmentCount":  it.EnrollmentCount,
			"checkinoutRecord": checkRecs,
		}
		programObj := map[string]interface{}{
			"id":            prog.ID.Hex(),
			"formId":        prog.FormID.Hex(),
			"name":          deref(prog.Name),
			"type":          prog.Type,
			"programState":  prog.ProgramState,
			"skill":         prog.Skill,
			"endDateEnroll": prog.EndDateEnroll,
			"file":          prog.File,
			"foodVotes":     prog.FoodVotes,
			"programItems":  []map[string]interface{}{programItemObj},
		}
		enrollmentObj := map[string]interface{}{
			"id":               e.ID.Hex(),
			"registrationDate": e.RegistrationDate,
			"food":             e.Food,
		}
		base := map[string]interface{}{
			"program":    programObj,
			"enrollment": enrollmentObj,
		}

		if h, ok := latestByItem[e.ProgramItemID]; ok {
			// เข้าร่วมแล้ว: add หรือ no_change
			if h.ChangeType == "add" || h.ChangeType == "no_change" || h.HoursChange >= 0 && h.ChangeType == "" {
				m := mapsClone(base)
				m["changedAt"] = h.ChangedAt
				m["changeType"] = h.ChangeType
				res.Participated = append(res.Participated, m)
				continue
			}
			// ไม่ได้เข้าร่วม: remove
			if h.ChangeType == "remove" || h.HoursChange < 0 {
				m := mapsClone(base)
				m["changedAt"] = h.ChangedAt
				m["changeType"] = h.ChangeType
				res.Absent = append(res.Absent, m)
				continue
			}
			// อย่างอื่นถือว่ายังไม่เข้าร่วม
			res.Pending = append(res.Pending, base)
		} else {
			// ไม่มี history → ยังไม่เข้าร่วม
			res.Pending = append(res.Pending, base)
		}
	}

	return res, nil
}

// mapsClone คัดลอก map[string]interface{}
func mapsClone(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// deref คืนค่าสตริงจาก *string ถ้าว่างให้เป็น ""
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
func GetEnrollmentId(studentID, programItemID primitive.ObjectID) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var res struct {
		ID primitive.ObjectID `bson:"_id"`
	}

	err := DB.EnrollmentCollection.FindOne(
		ctx,
		bson.M{
			"studentId":     studentID,
			"programItemId": programItemID,
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
	ProgramItemID  string                  `json:"programItemId"`
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
func RegisterStudentsByCodes(ctx context.Context, programItemID primitive.ObjectID, items []BulkEnrollItem) (*BulkEnrollResult, error) {
	res := &BulkEnrollResult{
		ProgramItemID:  programItemID.Hex(),
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
		if err := RegisterStudent(programItemID, stu.ID, it.Food); err != nil {
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
func GetEnrollmentByProgramItemID(
	programItemID primitive.ObjectID,
	pagination models.PaginationParams,
	majors []string,
	status []int,
	studentYears []int,
	dateStr string,
) ([]bson.M, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1) pipeline เริ่มต้น
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"programItemId": programItemID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "Students",
			"localField":   "studentId",
			"foreignField": "_id",
			"as":           "student",
		}}},
		{{Key: "$unwind", Value: "$student"}},
		{{Key: "$lookup", Value: bson.M{
			"from": "Enrollments",
			"let":  bson.M{"studentId": "$student._id"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{
						"$and": bson.A{
							bson.M{"$eq": bson.A{"$studentId", "$$studentId"}},
							bson.M{"$eq": bson.A{"$programItemId", programItemID}},
						},
					},
				}}},
			},
			"as": "enrollment",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$enrollment",
			"preserveNullAndEmptyArrays": true,
		}}},
	}

	// 2) ฟิลเตอร์ (major/status/year/search)
	filter := bson.D{}
	if len(majors) > 0 {
		filter = append(filter, bson.E{Key: "student.major", Value: bson.M{"$in": majors}})
	}
	if len(status) > 0 {
		filter = append(filter, bson.E{Key: "student.status", Value: bson.M{"$in": status}})
	}
	if len(studentYears) > 0 {
		var regexFilters []bson.M
		for _, y := range programs.GenerateStudentCodeFilter(studentYears) {
			regexFilters = append(regexFilters, bson.M{"student.code": bson.M{"$regex": "^" + y, "$options": "i"}})
		}
		filter = append(filter, bson.E{Key: "$or", Value: regexFilters})
	}
	if s := strings.TrimSpace(pagination.Search); s != "" {
		re := bson.M{"$regex": s, "$options": "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.M{"student.code": re},
			bson.M{"student.name": re},
		}})
	}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: filter}})
	}

	// 3) Project + แปลง checkinoutRecord ให้พก programItemId (แบบเดียวกับอีกฟังก์ชัน)
	pipeline = append(pipeline, bson.D{{Key: "$project", Value: bson.M{
		"_id":              0,
		"id":               "$student._id",
		"code":             "$student.code",
		"name":             "$student.name",
		"engName":          "$student.engName",
		"status":           "$student.status",
		"softSkill":        "$student.softSkill",
		"hardSkill":        "$student.hardSkill",
		"major":            "$student.major",
		"enrollmentId":     "$enrollment._id",
		"food":             "$enrollment.food",
		"registrationDate": "$enrollment.registrationDate",
		"checkInOut": bson.M{
			"$map": bson.M{
				"input": bson.M{"$ifNull": bson.A{"$enrollment.checkinoutRecord", bson.A{}}},
				"as":    "r",
				"in": bson.M{
					"programItemId": "$programItemId",
					"r":             "$$r",
				},
			},
		},
		"checkInStatus": nil,
	}}})

	// 4) กรองรายวันด้วย timezone: "UTC" (เหมือนอีกตัว)
	if dateStr != "" {
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"checkInOut": bson.M{
				"$filter": bson.M{
					"input": "$checkInOut",
					"as":    "x",
					"cond": bson.M{"$eq": bson.A{
						bson.M{"$dateToString": bson.M{
							"format":   "%Y-%m-%d",
							"date":     bson.M{"$ifNull": bson.A{"$$x.r.checkin", "$$x.r.checkout"}},
							"timezone": "UTC",
						}},
						dateStr,
					}},
				},
			},
		}}})
	}

	// 5) นับ total
	countPipeline := append(append(mongo.Pipeline{}, pipeline...), bson.D{{Key: "$count", Value: "total"}})
	countCursor, err := DB.EnrollmentCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer countCursor.Close(ctx)

	var total int64
	if countCursor.Next(ctx) {
		var cr struct {
			Total int64 `bson:"total"`
		}
		if err := countCursor.Decode(&cr); err == nil {
			total = cr.Total
		}
	}

	// 6) ใส่ pagination แล้ว query จริง
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}
	pipeline = append(pipeline,
		bson.D{{Key: "$skip", Value: (pagination.Page - 1) * pagination.Limit}},
		bson.D{{Key: "$limit", Value: pagination.Limit}},
	)

	cursor, err := DB.EnrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	// 7) คำนวณ checkInStatus แบบเดิม (±15 นาทีจากเวลาเริ่ม) เฉพาะ "วันเป้าหมาย"
	loc, _ := time.LoadLocation("Asia/Bangkok")
	targetDate := dateStr
	if targetDate == "" {
		targetDate = time.Now().In(loc).Format("2006-01-02")
	}

	// ดึง ProgramItem (อันเดียว เพราะเป็น byProgramItemID)
	var item models.ProgramItem
	if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": programItemID}).Decode(&item); err == nil {
		// หา start time ของ targetDate
		var start time.Time
		hasStart := false
		for _, d := range item.Dates {
			if d.Date == targetDate && d.Stime != "" {
				if st, e := time.ParseInLocation("2006-01-02 15:04", d.Date+" "+d.Stime, loc); e == nil {
					start = st
					hasStart = true
					break
				}
			}
		}

		if hasStart {
			for i := range results {
				statusTxt := "ยังไม่เช็คชื่อ"

				// checkInOut: [{ programItemId, r: {checkin, checkout, participation} }, ...]
				if arr, ok := results[i]["checkInOut"].(primitive.A); ok {
					for _, v := range arr {
						m, ok := v.(bson.M)
						if !ok {
							continue
						}
						r, _ := m["r"].(bson.M)
						if r == nil {
							continue
						}
						if t, ok2 := r["checkin"].(primitive.DateTime); ok2 {
							tin := t.Time().In(loc)
							if tin.Format("2006-01-02") != targetDate {
								continue
							}
							early := start.Add(-15 * time.Minute)
							late := start.Add(15 * time.Minute)
							if (tin.Equal(early) || tin.After(early)) && (tin.Before(late) || tin.Equal(late)) {
								statusTxt = "ตรงเวลา"
							} else {
								statusTxt = "สาย"
							}
							break
						}
					}
				}

				results[i]["checkInStatus"] = statusTxt
			}
		} else {
			// ไม่มีเวลาเริ่มของวันนั้น → ให้สถานะว่างไว้/ยังไม่เช็คชื่อ
			for i := range results {
				results[i]["checkInStatus"] = "ยังไม่เช็คชื่อ"
			}
		}
	}

	return results, total, nil
}

func GetEnrollmentsByProgramID(
	programID primitive.ObjectID,
	pagination models.PaginationParams,
	majors []string,
	status []int,
	studentYears []int,
	dateStr string,
) ([]bson.M, int64, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	log.Println("dateStr", dateStr)
	// 1) ดึง programItemIds ทั้งหมดของโปรแกรมนี้
	itemCur, err := DB.ProgramItemCollection.Find(
		ctx,
		bson.M{"programId": programID},
		options.Find().SetProjection(bson.M{"_id": 1}),
	)
	if err != nil {
		return nil, 0, err
	}
	defer itemCur.Close(ctx)

	var itemIDs []primitive.ObjectID
	for itemCur.Next(ctx) {
		var v struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := itemCur.Decode(&v); err == nil {
			itemIDs = append(itemIDs, v.ID)
		}
	}
	if len(itemIDs) == 0 {
		return []bson.M{}, 0, nil
	}

	// 2) สร้าง pipeline หลัก (collection = Enrollments)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"programItemId": bson.M{"$in": itemIDs}}}},

		// join student
		{{Key: "$lookup", Value: bson.M{
			"from":         "Students",
			"localField":   "studentId",
			"foreignField": "_id",
			"as":           "student",
		}}},
		{{Key: "$unwind", Value: "$student"}},

		// project ฟิลด์ที่ต้องใช้ + แปลง checkinoutRecord ให้ "พก programItemId ติดไปกับทุกรายการ"
		// checkInOut = [{ programItemId, r: {checkin, checkout, participation} }, ...]
		{{Key: "$project", Value: bson.M{
			"_id":              0,
			"studentId":        "$student._id",
			"code":             "$student.code",
			"name":             "$student.name",
			"engName":          "$student.engName",
			"status":           "$student.status",
			"softSkill":        "$student.softSkill",
			"hardSkill":        "$student.hardSkill",
			"major":            "$student.major",
			"enrollmentId":     "$_id",
			"food":             "$food",
			"registrationDate": "$registrationDate",
			"checkInOut": bson.M{
				"$map": bson.M{
					"input": bson.M{"$ifNull": bson.A{"$checkinoutRecord", bson.A{}}},
					"as":    "r",
					"in": bson.M{
						"programItemId": "$programItemId",
						"r":             "$$r",
					},
				},
			},

			"checkInStatus": nil, // คำนวณภายหลัง
		}}},
	}

	// 3) ฟิลเตอร์ (หลัง $project เพื่ออ้างฟิลด์ได้ง่าย)
	filter := bson.D{}
	if len(majors) > 0 {
		filter = append(filter, bson.E{Key: "major", Value: bson.M{"$in": majors}})
	}
	if len(status) > 0 {
		filter = append(filter, bson.E{Key: "status", Value: bson.M{"$in": status}})
	}
	if len(studentYears) > 0 {
		var regexFilters []bson.M
		for _, y := range programs.GenerateStudentCodeFilter(studentYears) {
			regexFilters = append(regexFilters, bson.M{"code": bson.M{"$regex": "^" + y, "$options": "i"}})
		}
		filter = append(filter, bson.E{Key: "$or", Value: regexFilters})
	}
	if s := strings.TrimSpace(pagination.Search); s != "" {
		regex := bson.M{"$regex": s, "$options": "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.M{"code": regex},
			bson.M{"name": regex},
		}})
	}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: filter}})
	}

	// 4) รวมเป็น "คนละ 1 แถว" แล้ว flatten checkInOut ของทุก enrollment
	pipeline = append(pipeline,
		bson.D{{Key: "$group", Value: bson.M{
			"_id":              "$studentId",
			"studentId":        bson.M{"$first": "$studentId"},
			"code":             bson.M{"$first": "$code"},
			"name":             bson.M{"$first": "$name"},
			"engName":          bson.M{"$first": "$engName"},
			"status":           bson.M{"$first": "$status"},
			"softSkill":        bson.M{"$first": "$softSkill"},
			"hardSkill":        bson.M{"$first": "$hardSkill"},
			"major":            bson.M{"$first": "$major"},
			"food":             bson.M{"$first": "$food"},
			"registrationDate": bson.M{"$min": "$registrationDate"},
			"enrollmentId":     bson.M{"$first": "$enrollmentId"},

			"checkInOutNested": bson.M{
				"$push": bson.M{"$ifNull": bson.A{"$checkInOut", bson.A{}}},
			},
		}}},

		bson.D{{Key: "$addFields", Value: bson.M{
			"checkInOut": bson.M{
				"$reduce": bson.M{
					"input":        bson.M{"$ifNull": bson.A{"$checkInOutNested", bson.A{}}},
					"initialValue": bson.A{},
					"in":           bson.M{"$concatArrays": bson.A{"$$value", "$$this"}},
				},
			},
		}}},

		bson.D{{Key: "$addFields", Value: bson.M{"id": "$_id"}}},
		bson.D{{Key: "$project", Value: bson.M{"_id": 0, "checkInOutNested": 0}}},
	)

	// 5) กรองรายวัน (ถ้ามี dateStr) — เทียบจาก r.checkin ถ้า null ให้ใช้ r.checkout
	if dateStr != "" {
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"checkInOut": bson.M{
				"$filter": bson.M{
					"input": "$checkInOut",
					"as":    "x",
					"cond": bson.M{"$eq": bson.A{
						bson.M{"$dateToString": bson.M{
							"format":   "%Y-%m-%d",
							"date":     bson.M{"$ifNull": bson.A{"$$x.r.checkin", "$$x.r.checkout"}},
							"timezone": "UTC", // << เปลี่ยนเป็น UTC
						}},
						dateStr,
					}},
				},
			},
		}}})
	}

	// 6) sort
	order := 1
	if strings.ToLower(pagination.Order) == "desc" {
		order = -1
	}
	sortDoc := bson.D{{Key: "code", Value: order}}
	switch pagination.SortBy {
	case "name":
		sortDoc = bson.D{{Key: "name", Value: order}}
	case "major":
		sortDoc = bson.D{{Key: "major", Value: order}}
	case "status":
		sortDoc = bson.D{{Key: "status", Value: order}}
	case "registrationDate":
		sortDoc = bson.D{{Key: "registrationDate", Value: order}}
	}
	pipeline = append(pipeline, bson.D{{Key: "$sort", Value: sortDoc}})

	// 7) นับ total ก่อนใส่ skip/limit
	countPipeline := append(mongo.Pipeline{}, pipeline...)
	countPipeline = append(countPipeline, bson.D{{Key: "$count", Value: "total"}})
	countCursor, err := DB.EnrollmentCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer countCursor.Close(ctx)

	var total int64
	if countCursor.Next(ctx) {
		var c struct {
			Total int64 `bson:"total"`
		}
		if err := countCursor.Decode(&c); err == nil {
			total = c.Total
		}
	}

	// 8) ใส่ pagination แล้ว query จริง
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}
	pipeline = append(pipeline,
		bson.D{{Key: "$skip", Value: (pagination.Page - 1) * pagination.Limit}},
		bson.D{{Key: "$limit", Value: pagination.Limit}},
	)

	cursor, err := DB.EnrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	// 9) คำนวณสถานะ "แบบเดิม" (±15 นาที) ต่อ "วันเป้าหมาย"
	loc, _ := time.LoadLocation("Asia/Bangkok")
	targetDate := dateStr
	if targetDate == "" {
		targetDate = time.Now().In(loc).Format("2006-01-02")
	}

	// cache programItemId -> start time ของ targetDate
	startTimeByItem := map[string]time.Time{}
	getStart := func(itemID primitive.ObjectID) (time.Time, bool) {
		key := itemID.Hex()
		if v, ok := startTimeByItem[key]; ok {
			return v, true
		}
		var item models.ProgramItem
		if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": itemID}).Decode(&item); err != nil {
			return time.Time{}, false
		}
		for _, d := range item.Dates {
			if d.Date == targetDate && d.Stime != "" {
				if st, e := time.ParseInLocation("2006-01-02 15:04", d.Date+" "+d.Stime, loc); e == nil {
					startTimeByItem[key] = st
					return st, true
				}
			}
		}
		return time.Time{}, false
	}

	for i := range results {
		statusTxt := "ยังไม่เช็คชื่อ"

		// checkInOut: [{ programItemId, r: {checkin, checkout, participation} }, ...]
		if arr, ok := results[i]["checkInOut"].(primitive.A); ok {
			for _, v := range arr {
				m, ok := v.(bson.M)
				if !ok {
					continue
				}
				itemID, _ := m["programItemId"].(primitive.ObjectID)
				r, _ := m["r"].(bson.M)
				if r == nil {
					continue
				}

				// ใช้ checkin เป็นหลักในการตัดสิน (เหมือนเดิม)
				if t, ok2 := r["checkin"].(primitive.DateTime); ok2 {
					// เทียบเฉพาะรายการของ targetDate (pipeline กรองมาแล้วถ้ามี dateStr)
					tin := t.Time().In(loc)
					if tin.Format("2006-01-02") != targetDate {
						continue
					}

					st, ok3 := getStart(itemID)
					if !ok3 {
						continue
					}

					early := st.Add(-15 * time.Minute)
					late := st.Add(15 * time.Minute)
					if (tin.Equal(early) || tin.After(early)) && (tin.Before(late) || tin.Equal(late)) {
						statusTxt = "ตรงเวลา"
					} else {
						statusTxt = "สาย"
					}
					break // เอา record แรกที่เจอของวันนั้นพอ
				}
			}
		}

		results[i]["checkInStatus"] = statusTxt
	}

	return results, total, nil
}
