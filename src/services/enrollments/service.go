package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	hourhistory "Backend-Bluelock-007/src/services/hour-history"
	"Backend-Bluelock-007/src/services/programs"
	"Backend-Bluelock-007/src/services/summary_reports"
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

// ดึงกิจกรรมทั้งหมดที่ Student ลงทะเบียนไปแล้ว พร้อม pagination และ filter
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

// Student ลงทะเบียนกิจกรรม (ลงซ้ำไม่ได้ + เช็ค major + กันเวลาทับซ้อน)
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

	fmt.Println("Before recording hour change history.................")

	// 10) 📝 บันทึก HourChangeHistory สำหรับ Enrollment
	var program models.Program
	if err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programItem.ProgramID}).Decode(&program); err == nil {
		programName := "Unknown Program"
		if program.Name != nil {
			programName = *program.Name
		}
		hours := 0

		if err := hourhistory.RecordEnrollmentHourChange(
			ctx,
			studentID,
			newEnrollment.ID,
			programItem.ProgramID,
			programName,
			program.Skill,
			hours,
		); err != nil {
			log.Printf("⚠️ Warning: Failed to record enrollment hour change: %v", err)
			// Don't return error - we don't want to fail enrollment if hour history fails
		}
	} else {
		log.Printf("⚠️ Warning: Failed to get program info for hour history: %v", err)
	}

	return nil
}
func RegisterStudentByAdmin(programItemID, studentID primitive.ObjectID, food *string) error {
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
	// var student models.Student
	// if err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student); err != nil {
	// 	if err == mongo.ErrNoDocuments {
	// 		return errors.New("student not found")
	// 	}
	// 	return err
	// }

	// ✅ เช็คสาขา: กิจกรรมอนุญาตเฉพาะบาง major
	// if len(programItem.Majors) > 0 {
	// 	allowed := false
	// 	for _, m := range programItem.Majors {
	// 		log.Println(programItem.Majors)
	// 		log.Println(student.Major)
	// 		if strings.EqualFold(m, student.Major) { // ปลอดภัยต่อเคสตัวพิมพ์เล็ก/ใหญ่
	// 			allowed = true
	// 			break
	// 		}
	// 	}
	// 	if !allowed {
	// 		return errors.New("ไม่สามารถลงทะเบียนได้: สาขาไม่ตรงกับเงื่อนไขของกิจกรรม")
	// 	}
	// }

	// (ถ้าต้องการเช็คชั้นปีด้วย ให้เพิ่มเงื่อนไขจาก programItem.StudentYears ที่นี่ได้)

	// 5) กันเต็มโควต้า
	// if programItem.MaxParticipants != nil && programItem.EnrollmentCount >= *programItem.MaxParticipants {
	// 	return errors.New("ไม่สามารถลงทะเบียนได้ เนื่องจากจำนวนผู้เข้าร่วมเต็มแล้ว")
	// }

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

	fmt.Println("Before recording hour change history.................")

	// 10) 📝 บันทึก HourChangeHistory สำหรับ Enrollment
	var program models.Program
	if err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programItem.ProgramID}).Decode(&program); err == nil {
		programName := "Unknown Program"
		if program.Name != nil {
			programName = *program.Name
		}
		hours := 0

		if err := hourhistory.RecordEnrollmentHourChange(
			ctx,
			studentID,
			newEnrollment.ID,
			programItem.ProgramID,
			programName,
			program.Skill,
			hours,
		); err != nil {
			log.Printf("⚠️ Warning: Failed to record enrollment hour change: %v", err)
			// Don't return error - we don't want to fail enrollment if hour history fails
		}
	} else {
		log.Printf("⚠️ Warning: Failed to get program info for hour history: %v", err)
	}

	return nil
}

func GetEnrollmentById(enrollmentID primitive.ObjectID) (*models.Enrollment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var enrollment models.Enrollment
	err := DB.EnrollmentCollection.FindOne(ctx, bson.M{"_id": enrollmentID}).Decode(&enrollment)
	if err != nil {
		return nil, err
	}

	return &enrollment, nil
}

// enrollments/service.go

func UpdateEnrollmentCheckinoutByRecordID(
	ctx context.Context,
	enrollmentID primitive.ObjectID,
	recordID primitive.ObjectID,
	checkinProvided bool, // ส่งฟิลด์นี้มาหรือไม่ (tri-state)
	checkin *time.Time, // อาจเป็น nil = ต้องการล้างค่า
	checkoutProvided bool, // ส่งฟิลด์นี้มาหรือไม่ (tri-state)
	checkout *time.Time, // อาจเป็น nil = ต้องการล้างค่า
) (*models.Enrollment, error) {

	// ---------- LOAD CURRENT ----------
	var current models.Enrollment
	if err := DB.EnrollmentCollection.FindOne(ctx, bson.M{"_id": enrollmentID}).Decode(&current); err != nil {
		return nil, err
	}

	var oldCin, oldCout *time.Time
	var targetRec *models.CheckinoutRecord
	if current.CheckinoutRecord != nil {
		for i := range *current.CheckinoutRecord {
			r := &(*current.CheckinoutRecord)[i]
			if r.ID == recordID {
				oldCin, oldCout = r.Checkin, r.Checkout
				targetRec = r
				break
			}
		}
	}
	if targetRec == nil {
		return nil, fmt.Errorf("record not found")
	}

	// ---------- LOAD PROGRAM ITEM ----------
	var item models.ProgramItem
	if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": current.ProgramItemID}).Decode(&item); err != nil {
		return nil, err
	}
	loc := bangkok()
	programID := item.ProgramID

	log.Printf("[upd] enrollmentID=%s recordID=%s programItemID=%s programID=%s tz=Asia/Bangkok",
		enrollmentID.Hex(), recordID.Hex(), current.ProgramItemID.Hex(), programID.Hex(),
	)
	log.Printf("[upd] oldCin=%v oldCout=%v oldPart=%q",
		oldCin, oldCout,
		func() string {
			if targetRec != nil && targetRec.Participation != nil {
				return *targetRec.Participation
			}
			return ""
		}(),
	)

	// ---------- EFFECTIVE VALUES (tri-state) ----------
	effCin := oldCin
	effCout := oldCout
	if checkinProvided {
		effCin = checkin // may be nil (clear)
	}
	if checkoutProvided {
		effCout = checkout // may be nil (clear)
	}
	log.Printf("[upd] effective cin=%v cout=%v", effCin, effCout)

	// ---------- VALIDATION: date must exist in Program_Items ----------
	if effCin != nil {
		day := effCin.In(loc).Format(fmtDay)
		if !dateExistsInItem(&item, day) {
			return nil, fmt.Errorf("ไม่อนุญาตตั้งค่า checkin: %s ไม่อยู่ในตารางกิจกรรม", day)
		}
		log.Printf("[upd] cin.day=%s isInProgramDates=true", day)
	}
	if effCout != nil {
		day := effCout.In(loc).Format(fmtDay)
		if !dateExistsInItem(&item, day) {
			return nil, fmt.Errorf("ไม่อนุญาตตั้งค่า checkout: %s ไม่อยู่ในตารางกิจกรรม", day)
		}
		log.Printf("[upd] cout.day=%s isInProgramDates=true", day)
	}

	// ---------- PARTICIPATION (single source of truth) ----------
	part := participationFor(&item, effCin, effCout, loc)
	if part != nil {
		log.Printf("[upd] newParticipation=%q", *part)
	} else {
		log.Printf("[upd] newParticipation=nil")
	}

	// ---------- BUILD $set ----------
	setFields := bson.M{}
	if checkinProvided {
		setFields["checkinoutRecord.$[el].checkin"] = checkin // allow nil to clear
	}
	if checkoutProvided {
		setFields["checkinoutRecord.$[el].checkout"] = checkout // allow nil to clear
	}
	if len(setFields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	setFields["checkinoutRecord.$[el].participation"] = part

	log.Printf("[upd] setFields keys=%v", func() []string {
		keys := make([]string, 0, len(setFields))
		for k := range setFields {
			keys = append(keys, k)
		}
		return keys
	}())

	// ---------- DB UPDATE ----------
	filter := bson.M{"_id": enrollmentID}
	update := bson.M{"$set": setFields}
	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After).
		SetArrayFilters(options.ArrayFilters{Filters: []interface{}{bson.M{"el._id": recordID}}})

	var updated models.Enrollment
	res := DB.EnrollmentCollection.FindOneAndUpdate(ctx, filter, update, opts)
	if err := res.Err(); err != nil {
		return nil, err
	}
	if err := res.Decode(&updated); err != nil {
		return nil, err
	}
	log.Printf("[upd] updated OK enrollmentID=%s", enrollmentID.Hex())

	// ---------- SUMMARY ADJUSTMENT ----------
	isLateFromParticipation := func(p *string) bool {
		if p == nil {
			return false
		}
		s := strings.TrimSpace(*p)

		// 1) จับ "ไม่ตรงเวลา" และเคสที่ไม่เข้าเกณฑ์ให้เป็น late ก่อน (ต้องมาก่อนคำว่า "ตรงเวลา")
		if strings.Contains(s, "ไม่ตรงเวลา") ||
			strings.Contains(s, "เวลาไม่เข้าเกณฑ์") ||
			strings.Contains(s, "ไม่พบเวลาเริ่ม") ||
			strings.Contains(s, "เช็คเอาท์อย่างเดียว") ||
			strings.Contains(s, "สาย") {
			return true
		}

		// 2) on-time เท่าที่ยอมรับ (เฉพาะข้อความที่ตรง)
		if s == "เช็คอิน/เช็คเอาท์ตรงเวลา" ||
			strings.Contains(s, "รอเช็คเอาท์") { // "เช็คอินแล้ว (รอเช็คเอาท์)"
			return false
		}

		// 3) เผื่อข้อความอื่น ๆ ที่ยังไม่รู้จัก: ถือว่าไม่ late ไว้ก่อน
		return false
	}

	var oldCinDay, newCinDay, oldCoutDay, newCoutDay string
	if oldCin != nil {
		oldCinDay = oldCin.In(loc).Format(fmtDay)
	}
	if effCin != nil {
		newCinDay = effCin.In(loc).Format(fmtDay)
	}
	if oldCout != nil {
		oldCoutDay = oldCout.In(loc).Format(fmtDay)
	}
	if effCout != nil {
		newCoutDay = effCout.In(loc).Format(fmtDay)
	}
	log.Printf("[sum] days oldCinDay=%s newCinDay=%s oldCoutDay=%s newCoutDay=%s",
		oldCinDay, newCinDay, oldCoutDay, newCoutDay,
	)

	oldCinLate := isLateFromParticipation(func() *string {
		if targetRec != nil {
			return targetRec.Participation
		}
		return nil
	}())
	newCinLate := isLateFromParticipation(part)
	log.Printf("[sum] cin late: old=%t new=%t", oldCinLate, newCinLate)

	// -------- Check-in --------
	switch {
	case oldCin != nil && effCin != nil && oldCinDay == newCinDay:
		log.Printf("[sum] cin same-day change oldLate=%t newLate=%t day=%s", oldCinLate, newCinLate, newCinDay)
		_ = summary_reports.EnsureSummaryReportExistsForDate(programID, newCinDay)
		if oldCinLate != newCinLate {
			log.Printf("[sum] cin moveBucket day=%s fromLate=%t toLate=%t (-1,+1)", newCinDay, oldCinLate, newCinLate)
			if err := summary_reports.AdjustCheckinCount(programID, newCinDay, -1, oldCinLate); err != nil {
				log.Printf("[sum][ERR] AdjustCheckinCount -1 day=%s late=%t err=%v", newCinDay, oldCinLate, err)
			}
			if err := summary_reports.AdjustCheckinCount(programID, newCinDay, 1, newCinLate); err != nil {
				log.Printf("[sum][ERR] AdjustCheckinCount +1 day=%s late=%t err=%v", newCinDay, newCinLate, err)
			}
			if err := summary_reports.RecalculateNotParticipating(programID, newCinDay); err != nil {
				log.Printf("[sum][ERR] RecalcNotParticipating day=%s err=%v", newCinDay, err)
			}
		} else {
			log.Printf("[sum] cin same-day noBucketChange")
		}

	case oldCin != nil && effCin == nil:
		log.Printf("[sum] cin clear day=%s late=%t (-1)", oldCinDay, oldCinLate)
		_ = summary_reports.EnsureSummaryReportExistsForDate(programID, oldCinDay)
		if err := summary_reports.AdjustCheckinCount(programID, oldCinDay, -1, oldCinLate); err != nil {
			log.Printf("[sum][ERR] AdjustCheckinCount -1 day=%s late=%t err=%v", oldCinDay, oldCinLate, err)
		}
		if err := summary_reports.RecalculateNotParticipating(programID, oldCinDay); err != nil {
			log.Printf("[sum][ERR] RecalcNotParticipating day=%s err=%v", oldCinDay, err)
		}

	case oldCin == nil && effCin != nil:
		log.Printf("[sum] cin add day=%s late=%t (+1)", newCinDay, newCinLate)
		_ = summary_reports.EnsureSummaryReportExistsForDate(programID, newCinDay)
		if err := summary_reports.AdjustCheckinCount(programID, newCinDay, 1, newCinLate); err != nil {
			log.Printf("[sum][ERR] AdjustCheckinCount +1 day=%s late=%t err=%v", newCinDay, newCinLate, err)
		}
		if err := summary_reports.RecalculateNotParticipating(programID, newCinDay); err != nil {
			log.Printf("[sum][ERR] RecalcNotParticipating day=%s err=%v", newCinDay, err)
		}

	case oldCin != nil && effCin != nil && oldCinDay != newCinDay:
		log.Printf("[sum] cin moveDay from=%s(late=%t) to=%s(late=%t) (-1,+1)", oldCinDay, oldCinLate, newCinDay, newCinLate)
		_ = summary_reports.EnsureSummaryReportExistsForDate(programID, oldCinDay)
		if err := summary_reports.AdjustCheckinCount(programID, oldCinDay, -1, oldCinLate); err != nil {
			log.Printf("[sum][ERR] AdjustCheckinCount -1 day=%s late=%t err=%v", oldCinDay, oldCinLate, err)
		}
		if err := summary_reports.RecalculateNotParticipating(programID, oldCinDay); err != nil {
			log.Printf("[sum][ERR] RecalcNotParticipating day=%s err=%v", oldCinDay, err)
		}
		_ = summary_reports.EnsureSummaryReportExistsForDate(programID, newCinDay)
		if err := summary_reports.AdjustCheckinCount(programID, newCinDay, 1, newCinLate); err != nil {
			log.Printf("[sum][ERR] AdjustCheckinCount +1 day=%s late=%t err=%v", newCinDay, newCinLate, err)
		}
		if err := summary_reports.RecalculateNotParticipating(programID, newCinDay); err != nil {
			log.Printf("[sum][ERR] RecalcNotParticipating day=%s err=%v", newCinDay, err)
		}
	}

	// -------- Check-out (no late bucket) --------
	switch {
	case oldCout != nil && effCout != nil && oldCoutDay == newCoutDay:
		log.Printf("[sum] cout same-day change day=%s (no bucket change)", newCoutDay)
		_ = summary_reports.EnsureSummaryReportExistsForDate(programID, newCoutDay)

	case oldCout != nil && effCout == nil:
		log.Printf("[sum] cout clear day=%s (-1)", oldCoutDay)
		_ = summary_reports.EnsureSummaryReportExistsForDate(programID, oldCoutDay)
		if err := summary_reports.AdjustCheckoutCount(programID, oldCoutDay, -1); err != nil {
			log.Printf("[sum][ERR] AdjustCheckoutCount -1 day=%s err=%v", oldCoutDay, err)
		}

	case oldCout == nil && effCout != nil:
		log.Printf("[sum] cout add day=%s (+1)", newCoutDay)
		_ = summary_reports.EnsureSummaryReportExistsForDate(programID, newCoutDay)
		if err := summary_reports.AdjustCheckoutCount(programID, newCoutDay, 1); err != nil {
			log.Printf("[sum][ERR] AdjustCheckoutCount +1 day=%s err=%v", newCoutDay, err)
		}

	case oldCout != nil && effCout != nil && oldCoutDay != newCoutDay:
		log.Printf("[sum] cout moveDay from=%s to=%s (-1,+1)", oldCoutDay, newCoutDay)
		_ = summary_reports.EnsureSummaryReportExistsForDate(programID, oldCoutDay)
		if err := summary_reports.AdjustCheckoutCount(programID, oldCoutDay, -1); err != nil {
			log.Printf("[sum][ERR] AdjustCheckoutCount -1 day=%s err=%v", oldCoutDay, err)
		}
		_ = summary_reports.EnsureSummaryReportExistsForDate(programID, newCoutDay)
		if err := summary_reports.AdjustCheckoutCount(programID, newCoutDay, 1); err != nil {
			log.Printf("[sum][ERR] AdjustCheckoutCount +1 day=%s err=%v", newCoutDay, err)
		}
	}

	return &updated, nil
}

// ยกเลิกการลงทะเบียน
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

	// ✅ ลบประวัติการเปลี่ยนแปลงชั่วโมงที่เกี่ยวข้องกับ enrollment นี้
	_, err = DB.HourChangeHistoryCollection.DeleteMany(ctx, bson.M{"enrollmentId": enrollmentID})
	if err != nil {
		log.Printf("⚠️ Warning: Failed to delete hour change histories for enrollmentId %s: %v", enrollmentID.Hex(), err)
		// Don't return error - we don't want to fail unenrollment if history deletion fails
	}

	return nil
}

// ดึงข้อมูลเฉพาะ Program ที่ Student ลงทะเบียนไว้ (1 ตัว)
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

	// 1) pipeline เริ่มต้น + joins
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
		{{Key: "$unwind", Value: bson.M{"path": "$enrollment", "preserveNullAndEmptyArrays": true}}},
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
		var ors []bson.M
		for _, y := range programs.GenerateStudentCodeFilter(studentYears) {
			ors = append(ors, bson.M{"student.code": bson.M{"$regex": "^" + y, "$options": "i"}})
		}
		filter = append(filter, bson.E{Key: "$or", Value: ors})
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

	// 3) Project raw (กัน null) + meta fields
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
		"rawCheckInOut":    bson.M{"$ifNull": bson.A{"$enrollment.checkinoutRecord", bson.A{}}},
	}}})

	// 4) กรองรายวัน (TZ = Asia/Bangkok)
	if dateStr != "" {
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"rawCheckInOut": bson.M{
				"$filter": bson.M{
					"input": "$rawCheckInOut",
					"as":    "r",
					"cond": bson.M{"$eq": bson.A{
						bson.M{"$dateToString": bson.M{
							"format":   "%Y-%m-%d",
							"date":     bson.M{"$ifNull": bson.A{"$$r.checkin", "$$r.checkout"}},
							"timezone": tzBangkok,
						}},
						dateStr,
					}},
				},
			},
		}}})
	}

	// 5) สร้าง checkInOut (string เวลาไทย +0700)
	pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
		"checkInOut": bson.M{"$map": bson.M{
			"input": "$rawCheckInOut",
			"as":    "r",
			"in": bson.M{
				"id": "$$r._id", // <<< เพิ่มบรรทัดนี้
				"checkin": bson.M{"$cond": bson.A{
					bson.M{"$ne": bson.A{"$$r.checkin", nil}},
					bson.M{"$dateToString": bson.M{
						"format":   mongoFmtISOOffset,
						"date":     "$$r.checkin",
						"timezone": tzBangkok,
					}},
					nil,
				}},
				"checkout": bson.M{"$cond": bson.A{
					bson.M{"$ne": bson.A{"$$r.checkout", nil}},
					bson.M{"$dateToString": bson.M{
						"format":   mongoFmtISOOffset,
						"date":     "$$r.checkout",
						"timezone": tzBangkok,
					}},
					nil,
				}},
				"participation": "$$r.participation",
			},
		}},
	}}})

	// 6) นับ total
	countPipeline := append(append(mongo.Pipeline{}, pipeline...), bson.D{{Key: "$count", Value: "total"}})
	countCur, err := DB.EnrollmentCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer countCur.Close(ctx)
	var total int64
	if countCur.Next(ctx) {
		var cr struct {
			Total int64 `bson:"total"`
		}
		_ = countCur.Decode(&cr)
		total = cr.Total
	}

	// 7) ใส่ pagination และ query
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

	cur, err := DB.EnrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var results []bson.M
	if err := cur.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	// 8) คำนวณ checkInStatus แบบเดิม (±15 นาที) อิง ProgramItem
	// loc := bangkok()
	// target := dateStr
	// if target == "" {
	// 	target = time.Now().In(loc).Format(fmtDay)
	// }

	// var item models.ProgramItem
	// if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": programItemID}).Decode(&item); err == nil {
	// 	var start time.Time
	// 	ok := false
	// 	for _, d := range item.Dates {
	// 		if d.Date == target && d.Stime != "" {
	// 			if st, e := time.ParseInLocation(fmtDay+" 15:04", d.Date+" "+d.Stime, loc); e == nil {
	// 				start, ok = st, true
	// 				break
	// 			}
	// 		}
	// 	}

	// 	for i := range results {
	// 		statusTxt := "ยังไม่เช็คชื่อ"

	// 		if ok {
	// 			if arr, okArr := results[i]["rawCheckInOut"].(primitive.A); okArr {
	// 				for _, v := range arr {
	// 					if r, okR := v.(bson.M); okR {
	// 						if dt, okDT := r["checkin"].(primitive.DateTime); okDT {
	// 							tin := dt.Time().In(loc)
	// 							if tin.Format(fmtDay) != target {
	// 								continue
	// 							}
	// 							early := start.Add(-15 * time.Minute)
	// 							late := start.Add(15 * time.Minute)
	// 							if (tin.Equal(early) || tin.After(early)) && (tin.Before(late) || tin.Equal(late)) {
	// 								statusTxt = "ตรงเวลา"
	// 							} else {
	// 								statusTxt = "สาย"
	// 							}
	// 							break
	// 						}
	// 					}
	// 				}
	// 			}
	// 		}

	// 		results[i]["checkInStatus"] = statusTxt
	// 		delete(results[i], "rawCheckInOut") // ลบ raw ก่อนส่งออก
	// 	}
	// }

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

	// 1) หา item ทั้งหมดของโปรแกรม
	itemCur, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID}, options.Find().SetProjection(bson.M{"_id": 1}))
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

	// 2) pipeline หลัก
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"programItemId": bson.M{"$in": itemIDs}}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "Students",
			"localField":   "studentId",
			"foreignField": "_id",
			"as":           "student",
		}}},
		{{Key: "$unwind", Value: "$student"}},
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

			// rawCheckInOut ต่อ item พร้อม programItemId
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
		}}},
	}

	// 3) ฟิลเตอร์ทั่วไป
	filter := bson.D{}
	if len(majors) > 0 {
		filter = append(filter, bson.E{Key: "major", Value: bson.M{"$in": majors}})
	}
	if len(status) > 0 {
		filter = append(filter, bson.E{Key: "status", Value: bson.M{"$in": status}})
	}
	if len(studentYears) > 0 {
		var ors []bson.M
		for _, y := range programs.GenerateStudentCodeFilter(studentYears) {
			ors = append(ors, bson.M{"code": bson.M{"$regex": "^" + y, "$options": "i"}})
		}
		filter = append(filter, bson.E{Key: "$or", Value: ors})
	}
	if s := strings.TrimSpace(pagination.Search); s != "" {
		re := bson.M{"$regex": s, "$options": "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.M{"code": re},
			bson.M{"name": re},
		}})
	}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: filter}})
	}

	// 4) รวมคนละ 1 แถว + flatten
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
			"checkInOutNested": bson.M{"$push": bson.M{"$ifNull": bson.A{"$checkInOut", bson.A{}}}},
		}}},
		bson.D{{Key: "$addFields", Value: bson.M{
			"rawCheckInOut": bson.M{
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

	// 5) กรองรายวัน (TZ = Asia/Bangkok)
	if dateStr != "" {
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"rawCheckInOut": bson.M{
				"$filter": bson.M{
					"input": "$rawCheckInOut",
					"as":    "x",
					"cond": bson.M{"$eq": bson.A{
						bson.M{"$dateToString": bson.M{
							"format":   "%Y-%m-%d",
							"date":     bson.M{"$ifNull": bson.A{"$$x.r.checkin", "$$x.r.checkout"}},
							"timezone": tzBangkok,
						}},
						dateStr,
					}},
				},
			},
		}}})
	}

	// 6) สร้าง checkInOut สำหรับแสดงผล (string เวลาไทย)
	pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
		"checkInOut": bson.M{
			"$map": bson.M{
				"input": "$rawCheckInOut",
				"as":    "x",
				"in": bson.M{
					"id": "$$x.r._id", // <<< เพิ่ม
					"checkin": bson.M{"$cond": bson.A{
						bson.M{"$ne": bson.A{"$$x.r.checkin", nil}},
						bson.M{"$dateToString": bson.M{
							"format":   mongoFmtISOOffset,
							"date":     "$$x.r.checkin",
							"timezone": tzBangkok,
						}},
						nil,
					}},
					"checkout": bson.M{"$cond": bson.A{
						bson.M{"$ne": bson.A{"$$x.r.checkout", nil}},
						bson.M{"$dateToString": bson.M{
							"format":   mongoFmtISOOffset,
							"date":     "$$x.r.checkout",
							"timezone": tzBangkok,
						}},
						nil,
					}},
					"participation": "$$x.r.participation",
				},
			},
		},
	}}})

	// 7) sort + count
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

	countPipeline := append(mongo.Pipeline{}, pipeline...)
	countPipeline = append(countPipeline, bson.D{{Key: "$count", Value: "total"}})
	countCur, err := DB.EnrollmentCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer countCur.Close(ctx)

	var total int64
	if countCur.Next(ctx) {
		var c struct {
			Total int64 `bson:"total"`
		}
		_ = countCur.Decode(&c)
		total = c.Total
	}

	// 8) pagination + query
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

	cur, err := DB.EnrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var results []bson.M
	if err := cur.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	// // 9) คำนวณสถานะแบบเดิม (±15 นาที) ต่อวัน โดย ProgramItem เป็นหลัก
	// loc := bangkok()
	// target := dateStr
	// if target == "" {
	// 	target = time.Now().In(loc).Format(fmtDay)
	// }

	// // cache start time ต่อ programItem
	// startTimeByItem := map[string]time.Time{}
	// getStart := func(itemID primitive.ObjectID) (time.Time, bool) {
	// 	key := itemID.Hex()
	// 	if v, ok := startTimeByItem[key]; ok {
	// 		return v, true
	// 	}
	// 	var item models.ProgramItem
	// 	if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": itemID}).Decode(&item); err != nil {
	// 		return time.Time{}, false
	// 	}
	// 	for _, d := range item.Dates {
	// 		if d.Date == target && d.Stime != "" {
	// 			if st, e := time.ParseInLocation(fmtDay+" 15:04", d.Date+" "+d.Stime, loc); e == nil {
	// 				startTimeByItem[key] = st
	// 				return st, true
	// 			}
	// 		}
	// 	}
	// 	return time.Time{}, false
	// }

	// for i := range results {
	// 	statusTxt := "ยังไม่เช็คชื่อ"

	// 	if arr, ok := results[i]["rawCheckInOut"].(primitive.A); ok {
	// 		for _, v := range arr {
	// 			m, ok := v.(bson.M)
	// 			if !ok {
	// 				continue
	// 			}
	// 			itemID, _ := m["programItemId"].(primitive.ObjectID)
	// 			r, _ := m["r"].(bson.M)
	// 			if r == nil {
	// 				continue
	// 			}
	// 			if dt, okDT := r["checkin"].(primitive.DateTime); okDT {
	// 				st, okStart := getStart(itemID)
	// 				if !okStart {
	// 					continue
	// 				}
	// 				tin := dt.Time().In(loc)
	// 				if tin.Format(fmtDay) != target {
	// 					continue
	// 				}
	// 				early := st.Add(-15 * time.Minute)
	// 				late := st.Add(15 * time.Minute)
	// 				if (tin.Equal(early) || tin.After(early)) && (tin.Before(late) || tin.Equal(late)) {
	// 					statusTxt = "ตรงเวลา"
	// 				} else {
	// 					statusTxt = "สาย"
	// 				}
	// 				break
	// 			}
	// 		}
	// 	}

	// 	results[i]["checkInStatus"] = statusTxt
	// 	delete(results[i], "rawCheckInOut") // ลบ raw ก่อนส่งออก
	// }

	return results, total, nil
}
