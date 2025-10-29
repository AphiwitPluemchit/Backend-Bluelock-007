package students

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	hourhistory "Backend-Bluelock-007/src/services/hour-history"
	"Backend-Bluelock-007/src/services/programs"
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// Collections are now initialized in service.go

// GetStudentsWithFilter - ดึงข้อมูลนิสิตทั้งหมดที่ผ่านการ filter ตามเงื่อนไขที่ระบุ
func GetStudentsWithFilter(params models.PaginationParams, majors []string, studentYears []string, studentStatus []string) ([]bson.M, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{}

	// 🔍 Search (name, code)
	if params.Search != "" {
		regex := bson.M{"$regex": params.Search, "$options": "i"}
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{
			"$or": bson.A{
				bson.M{"name": regex},
				bson.M{"code": regex},
			},
		}}})
	}

	// 🔍 Filter: major
	if len(majors) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{
			"major": bson.M{"$in": majors},
		}}})
	}

	// 🔍 Filter: status
	if len(studentStatus) > 0 {
		intStatus := make([]int, 0, len(studentStatus))
		for _, s := range studentStatus {
			if v, err := strconv.Atoi(s); err == nil {
				intStatus = append(intStatus, v)
			}
		}
		if len(intStatus) > 0 {
			pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{
				"status": bson.M{"$in": intStatus},
			}}})
		}
	}

	// 🔍 Filter: studentYears (prefix by code)
	if len(studentYears) > 0 {
		intYears := make([]int, 0, len(studentYears))
		for _, y := range studentYears {
			if v, err := strconv.Atoi(y); err == nil {
				intYears = append(intYears, v)
			}
		}
		if len(intYears) > 0 {
			yearPrefixes := programs.GenerateStudentCodeFilter(intYears)
			var regexFilters []bson.M
			for _, prefix := range yearPrefixes {
				regexFilters = append(regexFilters, bson.M{"code": bson.M{"$regex": "^" + prefix}})
			}
			pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{
				"$or": regexFilters,
			}}})
		}
	}

	// 🔢 Count (before pagination)
	countPipeline := append(append(mongo.Pipeline{}, pipeline...), bson.D{{Key: "$count", Value: "total"}})
	countCursor, err := DB.StudentCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	var countResult struct{ Total int64 `bson:"total"` }
	if countCursor.Next(ctx) {
		_ = countCursor.Decode(&countResult)
	}
	total := countResult.Total

	// 🔗 Lookup: users → email
	pipeline = append(pipeline, bson.D{{Key: "$lookup", Value: bson.M{
		"from":         "Users",
		"localField":   "_id",
		"foreignField": "refId",
		"as":           "user",
	}}})

	// 🔗 Lookup: Hour_Change_Histories → delta ต่อ skillType
	pipeline = append(pipeline, bson.D{{Key: "$lookup", Value: bson.M{
		"from": "Hour_Change_Histories",
		"let":  bson.M{"sid": "$_id"},
		"pipeline": mongo.Pipeline{
			// match ตาม studentId และสถานะที่นับจริง
			bson.D{{Key: "$match", Value: bson.M{
				"$expr": bson.M{"$eq": bson.A{"$studentId", "$$sid"}},
				"status": bson.M{"$in": bson.A{
					models.HCStatusAttended, models.HCStatusApproved, models.HCStatusAbsent,
				}},
			}}},
			// คำนวณ deltaHours (+abs attended/approved, -abs absent)
			bson.D{{Key: "$addFields", Value: bson.M{
				"deltaHours": bson.M{
					"$switch": bson.M{
						"branches": bson.A{
							bson.M{
								"case": bson.M{"$in": bson.A{"$status", bson.A{models.HCStatusAttended, models.HCStatusApproved}}},
								"then": bson.M{"$abs": bson.M{"$toInt": bson.M{"$ifNull": bson.A{"$hourChange", 0}}}},
							},
							bson.M{
								"case": bson.M{"$eq": bson.A{"$status", models.HCStatusAbsent}},
								"then": bson.M{
									"$multiply": bson.A{
										-1,
										bson.M{"$abs": bson.M{"$toInt": bson.M{"$ifNull": bson.A{"$hourChange", 0}}}},
									},
								},
							},
						},
						"default": 0,
					},
				},
			}}},
			// group รวมตาม skillType
			bson.D{{Key: "$group", Value: bson.M{
				"_id":        "$skillType", // "soft" | "hard"
				"totalHours": bson.M{"$sum": "$deltaHours"},
			}}},
			// รีเชปเป็น key-value แล้วรวมเป็น object {soft: X, hard: Y}
			bson.D{{Key: "$project", Value: bson.M{"k": "$_id", "v": "$totalHours", "_id": 0}}},
			bson.D{{Key: "$group", Value: bson.M{"_id": nil, "asMap": bson.M{"$push": bson.M{"k": "$k", "v": "$v"}}}}},
			bson.D{{Key: "$project", Value: bson.M{"_id": 0, "mapObj": bson.M{"$arrayToObject": "$asMap"}}}},
		},
		"as": "hourDeltaArr",
	}}})

	// 🔧 แตก softDelta / hardDelta ออกมา
	pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
		"softDelta": bson.M{
			"$ifNull": bson.A{
				bson.M{"$arrayElemAt": bson.A{"$hourDeltaArr.mapObj.soft", 0}},
				0,
			},
		},
		"hardDelta": bson.M{
			"$ifNull": bson.A{
				bson.M{"$arrayElemAt": bson.A{"$hourDeltaArr.mapObj.hard", 0}},
				0,
			},
		},
	}}})

	// 📌 Project: ใช้เฉพาะ delta จาก hour history (ไม่บวกกับ base hours จาก student)
	pipeline = append(pipeline, bson.D{{Key: "$project", Value: bson.M{
		"_id":     0,
		"id":      "$_id",
		"code":    1,
		"name":    1,
		"engName": 1,
		"status":  1,
		"major":   1,
		"email":   bson.M{"$arrayElemAt": bson.A{"$user.email", 0}},
		"softSkill": bson.M{"$ifNull": bson.A{"$softDelta", 0}},
		"hardSkill": bson.M{"$ifNull": bson.A{"$hardDelta", 0}},
	}}})

	// 🔁 Sort / Skip / Limit
	sort := 1
	if strings.ToLower(params.Order) == "desc" {
		sort = -1
	}
	sortBy := strings.TrimSpace(params.SortBy)
	if sortBy == "" {
		sortBy = "code" // default กัน null/empty
	}
	pipeline = append(pipeline,
		bson.D{{Key: "$sort", Value: bson.M{sortBy: sort}}},
		bson.D{{Key: "$skip", Value: (params.Page - 1) * params.Limit}},
		bson.D{{Key: "$limit", Value: params.Limit}},
	)

	// 🚀 Run
	cursor, err := DB.StudentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return results, total, totalPages, nil
}


// GetStudentByCode - ดึงข้อมูลนักศึกษาด้วยรหัส code พร้อม email และชั่วโมง soft/hard แบบสุทธิจาก HourChangeHistory
// func GetStudentByCode(code string) (bson.M, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	pipeline := mongo.Pipeline{
// 		{{Key: "$match", Value: bson.M{"code": code}}},

// 		{{Key: "$lookup", Value: bson.M{
// 			"from":         "Users",
// 			"localField":   "_id",
// 			"foreignField": "refId",
// 			"as":           "user",
// 		}}},

// 		// ⬇️ Lookup HourChangeHistory แล้วคำนวณ deltaHours (บวก/ลบ/0) ก่อนค่อย sum
// 		{{Key: "$lookup", Value: bson.M{
// 			"from": "HourChangeHistory",
// 			"let":  bson.M{"sid": "$_id"},
// 			"pipeline": mongo.Pipeline{
// 			  {{Key: "$match", Value: bson.M{
// 				"$expr": bson.M{"$eq": bson.A{"$studentId", "$$sid"}},
// 			  }}},
// 			  {{Key: "$addFields", Value: bson.M{
// 				"deltaHours": bson.M{
// 				  "$switch": bson.M{
// 					"branches": bson.A{
// 					  // บวก
// 					  bson.M{
// 						"case": bson.M{"$in": bson.A{"$status", bson.A{"attended", "approved"}}},
// 						"then": bson.M{"$toInt": bson.M{"$ifNull": bson.A{"$hourChange", 0}}},
// 					  },
// 					  // ลบ
// 					  bson.M{
// 						"case": bson.M{"$eq": bson.A{"$status", "absent"}},
// 						"then": bson.M{"$multiply": bson.A{-1, bson.M{"$toInt": bson.M{"$ifNull": bson.A{"$hourChange", 0}}}}},
// 					  },
// 					},
// 					"default": 0, // อื่น ๆ ไม่นับ
// 				  },
// 				},
// 			  }}},
// 			  {{Key: "$group", Value: bson.M{
// 				"_id":        "$skillType",            // "soft" | "hard"
// 				"totalHours": bson.M{"$sum": "$deltaHours"},
// 			  }}},
// 			},
// 			"as": "hourAgg",
// 		  }}},
		  
// 		  // map hourAgg -> {_hourMap.soft, _hourMap.hard} แล้วบวกกับค่า "ฐาน"
// 		  {{Key: "$addFields", Value: bson.M{
// 			"_hourMap": bson.M{
// 			  "$arrayToObject": bson.M{
// 				"$map": bson.M{
// 				  "input": "$hourAgg",
// 				  "as":    "h",
// 				  "in": bson.M{"k": "$$h._id", "v": "$$h.totalHours"},
// 				},
// 			  },
// 			},
// 		  }}},
// 		  {{Key: "$project", Value: bson.M{
// 			"_id": 0,
// 			"id":  "$_id",
// 			"code": 1, "name": 1, "engName": 1, "major": 1, "status": 1,
// 			"email": bson.M{"$arrayElemAt": bson.A{"$user.email", 0}},
// 			// ฐาน + ประวัติ (สุทธิ)
// 			"softSkill": bson.M{"$add": bson.A{bson.M{"$ifNull": bson.A{"$softSkill", 0}}, bson.M{"$ifNull": bson.A{"$_hourMap.soft", 0}}}},
// 			"hardSkill": bson.M{"$add": bson.A{bson.M{"$ifNull": bson.A{"$hardSkill", 0}}, bson.M{"$ifNull": bson.A{"$_hourMap.hard", 0}}}},
// 		  }}},
// 	}

// 	cursor, err := DB.StudentCollection.Aggregate(ctx, pipeline)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer cursor.Close(ctx)

// 	var results []bson.M
// 	if err := cursor.All(ctx, &results); err != nil {
// 		return nil, err
// 	}
// 	if len(results) == 0 {
// 		return nil, errors.New("student not found")
// 	}
// 	return results[0], nil
// }




func GetStudentById(id primitive.ObjectID) (*models.Student, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var student models.Student
	err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&student)
	if err != nil {
		return nil, err
	}
	return &student, nil
}

// ✅ ฟังก์ชันเข้ารหัส Password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// ✅ ตรวจสอบว่ามี Student ที่ `code` หรือ `email` ซ้ำกันหรือไม่
func isStudentExists(code string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := DB.StudentCollection.CountDocuments(ctx, bson.M{
		"$or": []bson.M{
			{"code": code},
		},
	})

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ✅ สร้าง Student พร้อมเพิ่ม User
func CreateStudent(userInput *models.User, studentInput *models.Student) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 ตรวจว่าซ้ำหรือไม่
	exists, err := isStudentExists(studentInput.Code)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("student already exists")
	}

	// ✅ เข้ารหัสรหัสผ่าน
	hashedPassword, err := hashPassword(userInput.Password)
	if err != nil {
		return errors.New("failed to hash password")
	}
	userInput.Password = hashedPassword

	// ✅ สร้าง student ก่อน
	studentInput.ID = primitive.NewObjectID()
	_, err = DB.StudentCollection.InsertOne(ctx, studentInput)
	if err != nil {
		return err
	}

	// ✅ สร้าง user โดยใช้ refId ไปยัง student
	userInput.ID = primitive.NewObjectID()
	userInput.Role = "Student"
	userInput.RefID = studentInput.ID // 👈 จุดสำคัญ
	userInput.Email = strings.ToLower(strings.TrimSpace(userInput.Email))
	userInput.IsActive = true

	_, err = DB.UserCollection.InsertOne(ctx, userInput)
	if err != nil {
		// rollback
		DB.StudentCollection.DeleteOne(ctx, bson.M{"_id": studentInput.ID})
		return err
	}

	return nil
}

// ✅ CreateOrUpdateStudent - สร้างหรืออัปเดต Student พร้อมจัดการ hour history สำหรับชั่วโมงจากระบบเก่า
func CreateOrUpdateStudent(userInput *models.User, studentInput *models.Student, legacySoftSkill, legacyHardSkill int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// ตรวจสอบว่า student มีอยู่แล้วหรือไม่
	var existingStudent models.Student
	err := DB.StudentCollection.FindOne(ctx, bson.M{"code": studentInput.Code}).Decode(&existingStudent)
	
	if err == mongo.ErrNoDocuments {
		// Student ไม่มีอยู่ - สร้างใหม่
		if err := createNewStudentWithHourHistory(ctx, userInput, studentInput, legacySoftSkill, legacyHardSkill); err != nil {
			return false, err
		}
		return true, nil // isNew = true
	} else if err != nil {
		return false, fmt.Errorf("error checking student existence: %v", err)
	}

	// Student มีอยู่แล้ว - อัปเดต
	if err := updateExistingStudentWithHourHistory(ctx, existingStudent.ID, userInput, studentInput, legacySoftSkill, legacyHardSkill); err != nil {
		return false, err
	}
	return false, nil // isNew = false
}

// helper function สำหรับสร้าง student ใหม่พร้อม hour history
func createNewStudentWithHourHistory(ctx context.Context, userInput *models.User, studentInput *models.Student, legacySoftSkill, legacyHardSkill int) error {
	// เข้ารหัสรหัสผ่าน
	hashedPassword, err := hashPassword(userInput.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}
	userInput.Password = hashedPassword

	// สร้าง student
	studentInput.ID = primitive.NewObjectID()
	// ไม่ใช้ soft/hard skill จาก studentInput แต่จะเก็บเป็น 0 เพราะจะใช้ hour history แทน
	studentInput.SoftSkill = 0
	studentInput.HardSkill = 0
	
	_, err = DB.StudentCollection.InsertOne(ctx, studentInput)
	if err != nil {
		return fmt.Errorf("failed to create student: %v", err)
	}

	// สร้าง user
	userInput.ID = primitive.NewObjectID()
	userInput.Role = "Student"
	userInput.RefID = studentInput.ID
	userInput.Email = strings.ToLower(strings.TrimSpace(userInput.Email))
	userInput.IsActive = true

	_, err = DB.UserCollection.InsertOne(ctx, userInput)
	if err != nil {
		// rollback student
		DB.StudentCollection.DeleteOne(ctx, bson.M{"_id": studentInput.ID})
		return fmt.Errorf("failed to create user: %v", err)
	}

	// สร้าง hour history สำหรับ soft skill (เสมอ แม้จะเป็น 0)
	if err := createLegacyHourHistory(ctx, studentInput.ID, "soft", legacySoftSkill); err != nil {
		log.Printf("Warning: Failed to create soft skill hour history for student %s: %v", studentInput.Code, err)
	}

	// สร้าง hour history สำหรับ hard skill (เสมอ แม้จะเป็น 0)
	if err := createLegacyHourHistory(ctx, studentInput.ID, "hard", legacyHardSkill); err != nil {
		log.Printf("Warning: Failed to create hard skill hour history for student %s: %v", studentInput.Code, err)
	}









 // ใช้ student ID เป็ source ID


	return nil
}

// createLegacyHourHistory - helper function สำหรับสร้าง hour history สำหรับ legacy import
func createLegacyHourHistory(ctx context.Context, studentID primitive.ObjectID, skillType string, hours int) error {
	skillTitle := "Soft Skill"
	if skillType == "hard" {
		skillTitle = "Hard Skill"
	}
	
	history := models.HourChangeHistory{
		ID:         primitive.NewObjectID(),
		SkillType:  skillType,
		Status:     models.HCStatusApproved,
		HourChange: hours,
		Remark:     "ชั่วโมงจากระบบเก่า",
		ChangeAt:   time.Now(),
		Title:      fmt.Sprintf("นำเข้าชั่วโมงจากระบบเก่า (%s)", skillTitle),
		StudentID:  studentID,
		SourceType: "legacy_import",
		SourceID:   studentID,
	}
	
	_, err := DB.HourChangeHistoryCollection.InsertOne(ctx, history)
	return err
}

// helper function สำหรับอัปเดต student ที่มีอยู่แล้วพร้อม hour history
func updateExistingStudentWithHourHistory(ctx context.Context, studentID primitive.ObjectID, userInput *models.User, studentInput *models.Student, legacySoftSkill, legacyHardSkill int) error {
	// อัปเดตข้อมูล student
	updateData := bson.M{
		"name":    studentInput.Name,
		"engName": studentInput.EngName,
		"major":   studentInput.Major,
		"status":  studentInput.Status,
	}
	
	_, err := DB.StudentCollection.UpdateOne(ctx, bson.M{"_id": studentID}, bson.M{"$set": updateData})
	if err != nil {
		return fmt.Errorf("failed to update student: %v", err)
	}

	// อัปเดต user
	_, err = DB.UserCollection.UpdateOne(ctx,
		bson.M{"refId": studentID, "role": "Student"},
		bson.M{"$set": bson.M{
			"name":  studentInput.Name,
			"email": userInput.Email,
		}})
	if err != nil {
		log.Printf("Warning: Failed to update user for student %v: %v", studentID, err)
	}

	// ลบ hour history เก่าที่มา sourceType = "legacy_import"
	_, err = DB.HourChangeHistoryCollection.DeleteMany(ctx, bson.M{
		"studentId":  studentID,
		"sourceType": "legacy_import",
	})
	if err != nil {
		log.Printf("Warning: Failed to delete old legacy hour history for student %v: %v", studentID, err)
	}

	// สร้าง hour history ใหม่โดยใช้ helper function
	if err := createLegacyHourHistory(ctx, studentID, "soft", legacySoftSkill); err != nil {
		log.Printf("Warning: Failed to create updated soft skill hour history for student %v: %v", studentID, err)
	}
	
	if err := createLegacyHourHistory(ctx, studentID, "hard", legacyHardSkill); err != nil {
		log.Printf("Warning: Failed to create updated hard skill hour history for student %v: %v", studentID, err)
	}

	return nil
}

// UpdateStudent - อัปเดตข้อมูล Student และ sync ไปยัง User
func UpdateStudent(id string, student *models.Student, email string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid student ID")
	}

	// ✅ อัปเดต student
	filter := bson.M{"_id": objID}
	update := bson.M{"$set": student}
	if _, err := DB.StudentCollection.UpdateOne(context.Background(), filter, update); err != nil {
		return err
	}

	// ✅ Sync ทั้ง name และ email ไปยัง user

	_, err = DB.UserCollection.UpdateOne(context.Background(),
		bson.M{"refId": objID, "role": "student"},
		bson.M{"$set": bson.M{
			"name":  student.Name,
			"email": email, // ✅ เพิ่ม email
		}})
	return err
}

// DeleteStudent - ลบ Student พร้อมลบ User ที่อ้างถึง
func DeleteStudent(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid student ID")
	}

	// ลบ user ที่ refId เป็น student.id และ role เป็น "student"
	_, err = DB.UserCollection.DeleteOne(context.Background(), bson.M{
		"refId": objID,
		"role":  "student",
	})
	if err != nil {
		return err
	}

	// ลบ student
	_, err = DB.StudentCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}

// UpdateStudentStatusByIDs - อัปเดตสถานะนักเรียนหลายคนโดยใช้ ID
func UpdateStudentStatusByIDs(studentIDs []string, status int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// แปลง string IDs เป็น ObjectIDs
	var objectIDs []primitive.ObjectID
	for _, id := range studentIDs {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fmt.Errorf("invalid student ID: %s", id)
		}
		objectIDs = append(objectIDs, objectID)
	}

	// อัปเดตสถานะนักเรียน
	filter := bson.M{"_id": bson.M{"$in": objectIDs}}
	update := bson.M{"$set": bson.M{"status": status}}

	result, err := DB.StudentCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update students: %v", err)
	}

	log.Printf("Updated %d students status to %d", result.ModifiedCount, status)

	// ✅ เฉพาะกรณีสถานะ = 0 (จัดเก็บ) เท่านั้นถึงจะอัปเดต isActive ใน users collection
	if status == 0 {
		userFilter := bson.M{"refId": bson.M{"$in": objectIDs}}
		userUpdate := bson.M{"$set": bson.M{"isActive": false}}

		userResult, err := DB.UserCollection.UpdateMany(ctx, userFilter, userUpdate)
		if err != nil {
			return fmt.Errorf("failed to update user isActive: %v", err)
		}

		log.Printf("Deactivated %d users linked to students", userResult.ModifiedCount)
	}

	return nil
}


// GetSammaryByCode - ดึงข้อมูลนักศึกษาด้วยรหัส code
func GetSammaryByCode(code string) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 ดึงข้อมูล student
	var student models.Student
	err := DB.StudentCollection.FindOne(ctx, bson.M{"code": code}).Decode(&student)
	if err != nil {
		return nil, errors.New("student not found")
	}

	// 🔄 Pipeline สำหรับหาประวัติ
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"studentId": student.ID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "Program_Items",
			"localField":   "programItemId",
			"foreignField": "_id",
			"as":           "programItem",
		}}},
		{{Key: "$unwind", Value: "$programItem"}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "Programs",
			"localField":   "programItem.programId",
			"foreignField": "_id",
			"as":           "program",
		}}},
		{{Key: "$unwind", Value: "$program"}},
		{{Key: "$project", Value: bson.M{
			"_id":              0,
			"registrationDate": "$registrationDate",
			"program": bson.M{
				"id":            "$program._id",
				"name":          "$program.name",
				"type":          "$program.type",
				"programState": "$program.programState",
				"skill":         "$program.skill",
				"programItem": bson.M{
					"id":          "$programItem._id",
					"name":        "$programItem.name",
					"dates":       "$programItem.dates",
					"hour":        "$programItem.hour",
					"operator":    "$programItem.operator",
					"description": "$programItem.description",
				},
			},
		}}},
	}

	cursor, err := DB.EnrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var history []bson.M
	if err := cursor.All(ctx, &history); err != nil {
		return nil, err
	}

	// ✅ return พร้อม history เต็ม
	return bson.M{
		"studentId": student.ID.Hex(),
		"code":      student.Code,
		"name":      student.Name,
		"major":     student.Major,
		"softSkill": student.SoftSkill,
		"hardSkill": student.HardSkill,
		"history":   history,
	}, nil
}

// GetSammaryByCodeWithHourHistory - ดึงข้อมูลนักศึกษาด้วยรหัส code พร้อมชั่วโมงสุทธิจาก HourChangeHistory
// ใช้ GetStudentWithCalculatedHours เป็น helper function ลดโค้ดซ้ำซ้อน
func GetSammaryByCodeWithHourHistory(code string) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1) ดึง student (ฐานชั่วโมง)
	var student models.Student
	if err := DB.StudentCollection.FindOne(ctx, bson.M{"code": code}).Decode(&student); err != nil {
		return nil, errors.New("student not found")
	}

	// 2) ใช้ helper function สำหรับคำนวณชั่วโมง (centralized logic)
	result, err := GetStudentWithCalculatedHours(ctx, student.ID)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ========================================
// Helper Functions - คำนวณชั่วโมงจาก Hour History
// ========================================

// GetStudentWithCalculatedHours - ดึงข้อมูล student พร้อมคำนวณชั่วโมงจาก hour history
// ฟังก์ชันนี้เป็น centralized function สำหรับคำนวณชั่วโมงแบบเดียวกันทั้งระบบ
func GetStudentWithCalculatedHours(ctx context.Context, studentID primitive.ObjectID) (bson.M, error) {
	// 1) ดึงข้อมูล student
	var student models.Student
	if err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&student); err != nil {
		return nil, fmt.Errorf("student not found: %v", err)
	}

	// 2) ดึงอีเมลจาก Users collection
	var user models.User
	email := ""
	if err := DB.UserCollection.FindOne(ctx, bson.M{"refId": studentID}).Decode(&user); err == nil {
		email = user.Email
	}

	// 3) คำนวณชั่วโมงจาก hour history เท่านั้น (ไม่ใช้ base hours จาก student collection)
	softSkillHours, hardSkillHours, err := hourhistory.CalculateNetHours(ctx, studentID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate net hours: %v", err)
	}

	// 4) ส่งกลับข้อมูลที่คำนวณแล้ว
	return bson.M{
		"id":        student.ID.Hex(),
		"studentId": student.ID.Hex(),
		"code":      student.Code,
		"name":      student.Name,
		"major":     student.Major,
		"email":     email,
		"softSkill": softSkillHours, // ชั่วโมงที่คำนวณจาก hour history
		"hardSkill": hardSkillHours, // ชั่วโมงที่คำนวณจาก hour history
		"status":    student.Status,
	}, nil
}


// Summary struct สำหรับ response
type SkillSummary struct {
	Completed    int `json:"completed"`
	NotCompleted int `json:"notCompleted"`
	Progress     int `json:"progress"` // %
}

type StudentSummary struct {
	Total          int          `json:"total"`
	Completed      int          `json:"completed"`
	NotCompleted   int          `json:"notCompleted"`
	CompletionRate int          `json:"completionRate"` // %
	SoftSkill      SkillSummary `json:"softSkill"`
	HardSkill      SkillSummary `json:"hardSkill"`
}

// GetStudentSummary - summary ตาม format ที่ต้องการ (เฉพาะนักเรียนที่มี status ไม่ใช่ 0)
func GetStudentSummary(majors []string, studentYears []string) (StudentSummary, error) {
	const softSkillTarget = 30
	const hardSkillTarget = 12

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// ---------- Build filter ----------
	filter := bson.M{"status": bson.M{"$ne": 0}}

	if len(majors) > 0 {
		filter["major"] = bson.M{"$in": majors}
	}

	if len(studentYears) > 0 {
		intYears := make([]int, 0, len(studentYears))
		for _, y := range studentYears {
			if v, err := strconv.Atoi(y); err == nil {
				intYears = append(intYears, v)
			}
		}
		if len(intYears) > 0 {
			yearPrefixes := programs.GenerateStudentCodeFilter(intYears)
			regexFilters := make([]bson.M, 0, len(yearPrefixes))
			for _, prefix := range yearPrefixes {
				regexFilters = append(regexFilters, bson.M{"code": bson.M{"$regex": "^" + prefix}})
			}
			filter["$or"] = regexFilters
		}
	}

	// ---------- Fetch students ----------
	cur, err := DB.StudentCollection.Find(ctx, filter)
	if err != nil {
		return StudentSummary{}, err
	}
	defer cur.Close(ctx)

	var students []models.Student
	if err := cur.All(ctx, &students); err != nil {
		return StudentSummary{}, err
	}

	total := len(students)
	if total == 0 {
		// ไม่มีนักศึกษา: คืน summary ว่าง ๆ
		summary := StudentSummary{
			Total:          0,
			Completed:      0,
			NotCompleted:   0,
			CompletionRate: 0,
			SoftSkill:      SkillSummary{Completed: 0, NotCompleted: 0, Progress: 0},
			HardSkill:      SkillSummary{Completed: 0, NotCompleted: 0, Progress: 0},
		}
		log.Printf("Student Summary (Status != 0): %+v", summary)
		return summary, nil
	}

	// ---------- Collect student IDs ----------
	ids := make([]primitive.ObjectID, 0, total)
	for _, s := range students {
		ids = append(ids, s.ID)
	}

	// ---------- Aggregate deltas from Hour_Change_Histories ----------
	// NOTE: ให้แน่ใจว่า DB.HourChangeHistoryCollection ชี้คอลเลกชันชื่อถูกต้อง
	deltaPipe := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"studentId": bson.M{"$in": ids},
			"status": bson.M{"$in": bson.A{
				models.HCStatusAttended, models.HCStatusApproved, models.HCStatusAbsent,
			}},
		}}},
		// normalize skillType -> skillKey (lower-case)
		{{Key: "$addFields", Value: bson.M{
			"skillKey": bson.M{"$toLower": "$skillType"},
		}}},
		// compute deltaHours
		{{Key: "$addFields", Value: bson.M{
			"deltaHours": bson.M{
				"$switch": bson.M{
					"branches": bson.A{
						bson.M{
							"case": bson.M{"$in": bson.A{"$status", bson.A{models.HCStatusAttended, models.HCStatusApproved}}},
							"then": bson.M{"$abs": bson.M{"$toInt": bson.M{"$ifNull": bson.A{"$hourChange", 0}}}},
						},
						bson.M{
							"case": bson.M{"$eq": bson.A{"$status", models.HCStatusAbsent}},
							"then": bson.M{
								"$multiply": bson.A{
									-1,
									bson.M{"$abs": bson.M{"$toInt": bson.M{"$ifNull": bson.A{"$hourChange", 0}}}},
								},
							},
						},
					},
					"default": 0,
				},
			},
		}}},
		// group per (studentId, skillKey)
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"studentId": "$studentId",
				"skillKey":  "$skillKey", // "soft" | "hard"
			},
			"totalHours": bson.M{"$sum": "$deltaHours"},
		}}},
	}

	type deltaRow struct {
		ID struct {
			StudentID primitive.ObjectID `bson:"studentId"`
			SkillKey  string             `bson:"skillKey"`
		} `bson:"_id"`
		TotalHours int64 `bson:"totalHours"`
	}

	dc, err := DB.HourChangeHistoryCollection.Aggregate(ctx, deltaPipe)
	if err != nil {
		return StudentSummary{}, fmt.Errorf("aggregate hour deltas error: %v", err)
	}
	defer dc.Close(ctx)

	type pair struct{ soft, hard int64 }
	deltaMap := make(map[primitive.ObjectID]pair, total)

	for dc.Next(ctx) {
		var r deltaRow
		if err := dc.Decode(&r); err != nil {
			return StudentSummary{}, fmt.Errorf("decode delta row error: %v", err)
		}
		p := deltaMap[r.ID.StudentID]
		switch r.ID.SkillKey {
		case "soft":
			p.soft += r.TotalHours
		case "hard":
			p.hard += r.TotalHours
		}
		deltaMap[r.ID.StudentID] = p
	}
	if err := dc.Err(); err != nil {
		return StudentSummary{}, fmt.Errorf("cursor error: %v", err)
	}

	// ---------- Count completion using NET hours ----------
	completed := 0
	softCompleted := 0
	hardCompleted := 0

	for _, s := range students {
		d := deltaMap[s.ID]
		// คำนวณจาก hour history เท่านั้น (ไม่ใช้ base hours จาก student)
		netSoft := d.soft
		netHard := d.hard

		if netSoft >= int64(softSkillTarget) {
			softCompleted++
		}
		if netHard >= int64(hardSkillTarget) {
			hardCompleted++
		}
		if netSoft >= int64(softSkillTarget) && netHard >= int64(hardSkillTarget) {
			completed++
		}
	}

	notCompleted := total - completed

	summary := StudentSummary{
		Total:          total,
		Completed:      completed,
		NotCompleted:   notCompleted,
		CompletionRate: percent(completed, total),
		SoftSkill: SkillSummary{
			Completed:    softCompleted,
			NotCompleted: total - softCompleted,
			Progress:     percent(softCompleted, total),
		},
		HardSkill: SkillSummary{
			Completed:    hardCompleted,
			NotCompleted: total - hardCompleted,
			Progress:     percent(hardCompleted, total),
		},
	}
	log.Printf("Student Summary (NET hours, Status != 0): %+v", summary)
	return summary, nil
}


func percent(part, total int) int {
	if total == 0 {
		return 0
	}
	return int(float64(part) / float64(total) * 100)
}
func FindExistingCodes(codes []string) ([]string, error) {
	if len(codes) == 0 {
		return []string{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	cur, err := DB.StudentCollection.Find(ctx, bson.M{"code": bson.M{"$in": codes}}, 
		/* options.Find() */)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	existsSet := make(map[string]struct{})
	for cur.Next(ctx) {
		var row struct{ Code string `bson:"code"` }
		if err := cur.Decode(&row); err == nil && row.Code != "" {
			existsSet[row.Code] = struct{}{}
		}
	}
	exists := make([]string, 0, len(existsSet))
	for code := range existsSet {
		exists = append(exists, code)
	}
	return exists, nil
}

// UpdateStudentStatus - อัปเดตสถานะนักศึกษาจากชั่วโมงสุทธิ (รับ string ID)
func UpdateStudentStatus(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid student ID: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	// เรียกใช้ฟังก์ชันจาก hour-history package
	return hourhistory.UpdateStudentStatus(ctx, objID)
}

