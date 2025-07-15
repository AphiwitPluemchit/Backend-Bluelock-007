package activities

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func getLightweightActivitiesPipeline(
	filter bson.M,
	sortField string, sortOrder int, isSortNearest bool,
	skip int64, limit int64,
	majors []string, studentYears []int,
) mongo.Pipeline {

	// โหลด Timezone สำหรับ "Asia/Bangkok"
	// ควรมีการจัดการ error หากโหลดไม่ได้
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		// หากโหลด Timezone ไม่ได้ ให้ log error และอาจจะใช้ UTC แทน
		// หรือ panic ถ้าถือว่าเป็นข้อผิดพลาดร้ายแรงที่ไม่สามารถดำเนินต่อได้
		// สำหรับ Production ควร log error และหาวิธีจัดการที่เหมาะสม
		fmt.Printf("Error loading timezone 'Asia/Bangkok': %v. Using UTC instead.", err)
		loc = time.UTC // ใช้ UTC เป็น fallback
		// UTC คือเวลามาตรฐานโลกที่ใช้เป็นจุดอ้างอิงสำหรับทุกโซนเวลา เช่น เวลาประเทศไทยเร็วกว่า UTC อยู่ 7 ชั่วโมง (UTC+7)
	}

	today := time.Now().In(loc).Format("2006-01-02")

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},

		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "activityItems"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "activityItems"},
		}}},
	}

	// set activityItems into another field
	pipeline = append(pipeline, bson.D{{Key: "$set", Value: bson.D{
		{Key: "fullActivityItems", Value: "$activityItems"},
	}},
	})

	// ✅ กรอง majors
	if len(majors) > 0 && majors[0] != "" {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{
			{Key: "activityItems.majors", Value: bson.D{{Key: "$in", Value: majors}}},
		}}})
	}

	// ✅ กรอง studentYears
	if len(studentYears) > 0 && studentYears[0] != 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{
			{Key: "activityItems.studentYears", Value: bson.D{{Key: "$in", Value: studentYears}}},
		}}})
	}

	// ✅ เงื่อนไขถ้าอยากเรียงตามวันจัดที่ใกล้สุด
	if sortField == "dates" && isSortNearest {

		pipeline = append(pipeline,
			// Unwind activityItems
			bson.D{{Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$activityItems"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			}}},
			// Unwind activityItems.dates
			bson.D{{Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$activityItems.dates"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			}}},

			// Match เฉพาะวันในอนาคต
			bson.D{{Key: "$match", Value: bson.D{
				{Key: "activityItems.dates.date", Value: bson.D{{Key: "$gte", Value: today}}},
			}}},
			// Group เอาวันที่ใกล้ที่สุดต่อ activity
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$_id"},
				{Key: "nextDate", Value: bson.D{{Key: "$min", Value: "$activityItems.dates.date"}}}, // nextDate คือ array ของ
				{Key: "name", Value: bson.D{{Key: "$first", Value: "$name"}}},
				{Key: "activityState", Value: bson.D{{Key: "$first", Value: "$activityState"}}},
				{Key: "skill", Value: bson.D{{Key: "$first", Value: "$skill"}}},
				{Key: "file", Value: bson.D{{Key: "$first", Value: "$file"}}},
				{Key: "activityItems", Value: bson.D{{Key: "$first", Value: "$fullActivityItems"}}},
			}}},

			// Sort nextDate
			bson.D{{Key: "$sort", Value: bson.D{{Key: "nextDate", Value: sortOrder}}}},
		)
	} else if sortField == "dates" {
		// ✅ กรณีใช้ field ปกติ
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{
			{Key: sortField, Value: sortOrder},
		}}})
	}

	// ✅ pagination
	if skip > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: skip}})
	}
	if limit > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: limit}})
	}

	return pipeline
}

func GetOneActivityPipeline(activityID primitive.ObjectID) mongo.Pipeline {
	return mongo.Pipeline{
		// 1️⃣ Match เฉพาะ Activity ที่ต้องการ
		{{
			Key: "$match", Value: bson.D{
				{Key: "_id", Value: activityID},
			},
		}},

		// 🔗 Lookup ActivityItems ที่เกี่ยวข้อง
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "activityItems"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "activityItems"},
		}}},
	}
}

func GetActivityStatisticsPipeline(activityID primitive.ObjectID) mongo.Pipeline {
	return mongo.Pipeline{
		// 1️⃣ Match เฉพาะ ActivityItems ที่ต้องการ
		{{
			Key: "$match", Value: bson.M{
				"activityId": activityID,
			},
		}},

		// 2️⃣ Lookup Enrollments จาก collection enrollments
		{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "enrollments"},
				{Key: "localField", Value: "_id"},
				{Key: "foreignField", Value: "activityItemId"},
				{Key: "as", Value: "enrollments"},
			},
		}},

		// 3️⃣ Unwind Enrollments
		{{
			Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$enrollments"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		}},

		// 4️⃣ Lookup Students
		{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "students"},
				{Key: "localField", Value: "enrollments.studentId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "student"},
			},
		}},

		// 5️⃣ Unwind Students
		{{
			Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$student"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		}},

		// 6️⃣ Group ตาม ActivityItem และ Major
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: bson.D{
					{Key: "activityItemId", Value: "$_id"},
					{Key: "majorName", Value: "$student.major"},
				}},
				{Key: "activityItemName", Value: bson.D{{Key: "$first", Value: "$name"}}},
				{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
				{Key: "maxParticipants", Value: bson.D{{Key: "$first", Value: "$maxParticipants"}}},
			},
		}},

		// 9️⃣ Group ActivityItemSums
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$_id.activityItemId"},
				{Key: "activityItemName", Value: bson.D{{Key: "$first", Value: "$activityItemName"}}},
				{Key: "maxParticipants", Value: bson.D{{Key: "$first", Value: "$maxParticipants"}}},
				{Key: "totalRegistered", Value: bson.D{{Key: "$sum", Value: "$count"}}},
				{Key: "registeredByMajor", Value: bson.D{{
					Key: "$push", Value: bson.D{
						{Key: "majorName", Value: "$_id.majorName"},
						{Key: "count", Value: "$count"},
					},
				}}},
			},
		}},

		// 🔟 Group Final Result
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: nil},
				{Key: "maxParticipants", Value: bson.D{{Key: "$sum", Value: "$maxParticipants"}}},
				{Key: "totalRegistered", Value: bson.D{{Key: "$sum", Value: "$totalRegistered"}}},
				{Key: "activityItemSums", Value: bson.D{{Key: "$push", Value: bson.D{
					{Key: "activityItemName", Value: "$activityItemName"},
					{Key: "registeredByMajor", Value: "$registeredByMajor"},
				}}}},
			},
		}},

		// 11️⃣ Add field remainingSlots
		{{
			Key: "$addFields", Value: bson.D{
				{Key: "remainingSlots", Value: bson.D{{Key: "$subtract", Value: bson.A{"$maxParticipants", "$totalRegistered"}}}},
			},
		}},

		// 12️⃣ Project Final Output
		{{
			Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "maxParticipants", Value: 1},
				{Key: "totalRegistered", Value: 1},
				{Key: "remainingSlots", Value: 1},
				{Key: "activityItemSums", Value: 1},
			},
		}},
	}
}
func GetActivityItemIDsByActivityID(ctx context.Context, activityID primitive.ObjectID) ([]primitive.ObjectID, error) {
	var activityItems []models.ActivityItem
	filter := bson.M{"activityId": activityID}
	cursor, err := database.ActivityItemCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &activityItems); err != nil {
		return nil, err
	}

	var activityItemIDs []primitive.ObjectID
	for _, item := range activityItems {
		activityItemIDs = append(activityItemIDs, item.ID)
	}

	return activityItemIDs, nil
}
func GetEnrollmentByActivityItemID(
	activityItemID primitive.ObjectID,
	pagination models.PaginationParams,
	majors []string,
	status []int,
	studentYears []int,
) ([]bson.M, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Base aggregation pipeline
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"activityItemId": activityItemID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "students",
			"localField":   "studentId",
			"foreignField": "_id",
			"as":           "student",
		}}},
		{{Key: "$unwind", Value: "$student"}},
		{{Key: "$lookup", Value: bson.M{
			"from": "enrollments",
			"let":  bson.M{"studentId": "$student._id"},
			"pipeline": mongo.Pipeline{
				{{"$match", bson.M{
					"$expr": bson.M{
						"$and": bson.A{
							bson.M{"$eq": bson.A{"$studentId", "$$studentId"}},
							bson.M{"$eq": bson.A{"$activityItemId", activityItemID}},
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

	// Filters
	filter := bson.D{}
	if len(majors) > 0 {
		filter = append(filter, bson.E{Key: "student.major", Value: bson.M{"$in": majors}})
	}
	if len(status) > 0 {
		filter = append(filter, bson.E{Key: "student.status", Value: bson.M{"$in": status}})
	}
	if len(studentYears) > 0 {
		var regexFilters []bson.M
		for _, year := range GenerateStudentCodeFilter(studentYears) {
			regexFilters = append(regexFilters, bson.M{"student.code": bson.M{"$regex": "^" + year, "$options": "i"}})
		}
		filter = append(filter, bson.E{Key: "$or", Value: regexFilters})
	}
	if pagination.Search != "" {
		regex := bson.M{"$regex": pagination.Search, "$options": "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.M{"student.name": regex},
			bson.M{"student.code": regex},
		}})
	}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: filter}})
	}

	// Project student fields
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
		"food":             "$enrollment.food",
		"registrationDate": "$enrollment.registrationDate",
	}}})

	// Count total before skip/limit
	countPipeline := append(pipeline, bson.D{{Key: "$count", Value: "total"}})
	countCursor, err := database.EnrollmentCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer countCursor.Close(ctx)

	var total int64
	if countCursor.Next(ctx) {
		var countResult struct {
			Total int64 `bson:"total"`
		}
		if err := countCursor.Decode(&countResult); err == nil {
			total = countResult.Total
		}
	}

	// Add pagination
	pipeline = append(pipeline,
		bson.D{{Key: "$skip", Value: (pagination.Page - 1) * pagination.Limit}},
		bson.D{{Key: "$limit", Value: pagination.Limit}},
	)

	cursor, err := database.EnrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

func GetActivitiesPipeline(filter bson.M, sortField string, sortOrder int, skip int64, limit int64, majors []string, studentYears []int) mongo.Pipeline {
	pipeline := mongo.Pipeline{
		// 🔍 Match เฉพาะ Activity ที่ต้องการ
		{{Key: "$match", Value: filter}},

		// 🔗 Lookup ActivityItems ที่เกี่ยวข้อง
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "activityItems"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "activityItems"},
		}}},

		// 🔥 Unwind ActivityItems เพื่อให้สามารถกรองได้
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$activityItems"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},

		// 3️⃣ Lookup EnrollmentCount แทนที่จะดึงทั้ง array
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "enrollments"},
			{Key: "let", Value: bson.D{{Key: "itemId", Value: "$activityItems._id"}}}, // let คือ การประกาศตัวแปรใน pipeline
			{Key: "pipeline", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{
					{Key: "$expr", Value: bson.D{ // ใช้ $expr เพื่อใช้ตัวแปรที่ประกาศใน let
						{Key: "$eq", Value: bson.A{"$activityItemId", "$$itemId"}},
					}},
				}}},
				bson.D{{Key: "$count", Value: "count"}},
			}},
			{Key: "as", Value: "activityItems.enrollmentCountData"},
		}}},

		// 4️⃣ Add enrollmentCount field จาก enrollmentCountData
		{{Key: "$addFields", Value: bson.D{
			{Key: "activityItems.enrollmentCount", Value: bson.D{
				{Key: "$ifNull", Value: bson.A{bson.D{
					{Key: "$arrayElemAt", Value: bson.A{"$activityItems.enrollmentCountData.count", 0}},
				}, 0}},
			}},
		}}},
	}

	// ✅ กรองเฉพาะ Major ที่ต้องการ **ถ้ามีค่า major**
	if len(majors) > 0 && majors[0] != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "activityItems.majors", Value: bson.D{{Key: "$in", Value: majors}}},
			}},
		})
	} else {
		fmt.Println("Skipping majorName filtering")
	}

	// ✅ กรองเฉพาะ StudentYears ที่ต้องการ **ถ้ามีค่า studentYears**
	if len(studentYears) > 0 {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "activityItems.studentYears", Value: bson.D{{Key: "$in", Value: studentYears}}},
			}},
		})
	}

	// ✅ Group ActivityItems กลับเข้าไปใน Activity
	pipeline = append(pipeline, bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "name", Value: bson.D{{Key: "$first", Value: "$name"}}},
			{Key: "activityState", Value: bson.D{{Key: "$first", Value: "$activityState"}}},
			{Key: "skill", Value: bson.D{{Key: "$first", Value: "$skill"}}},
			{Key: "file", Value: bson.D{{Key: "$first", Value: "$file"}}},
			{Key: "activityItems", Value: bson.D{{Key: "$push", Value: "$activityItems"}}}, // เก็บ ActivityItems เป็น Array
		}},
	})

	// ✅ ตรวจสอบและเพิ่ม `$sort` เฉพาะกรณีที่ต้องใช้
	if sortField != "" && (sortOrder == 1 || sortOrder == -1) {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: sortField, Value: sortOrder}}}})
	}

	// ✅ ตรวจสอบและเพิ่ม `$skip` และ `$limit` เฉพาะกรณีที่ต้องใช้
	if skip > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: skip}})
	}
	if limit > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: limit}})
	}

	return pipeline
}

func GetAllActivityCalendarPipeline(startDateStr string, endDateStr string) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"dates": bson.M{
				"$elemMatch": bson.M{
					"date": bson.M{
						"$gte": startDateStr,
						"$lte": endDateStr,
					},
				},
			},
		}}},

		// lookup enrollment
		{{Key: "$lookup", Value: bson.M{
			"from": "enrollments",
			"let":  bson.M{"activityItemId": "$_id"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{
						"$eq": bson.A{"$activityItemId", "$$activityItemId"},
					},
				}}},
				{{Key: "$count", Value: "count"}},
			},
			"as": "enrollments",
		}}},

		// unwind enrollment
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$enrollments",
			"preserveNullAndEmptyArrays": true, // เพื่อให้กิจกรรมที่ไม่มีการลงทะเบียนยังคงแสดง
		}}},

		{{Key: "$addFields", Value: bson.M{
			"enrollmentCount": bson.M{
				"$ifNull": bson.A{"$enrollments.count", 0}, // ถ้าไม่มี enrollments ให้ใช้ 0
			},
		}},
		},

		// Stage C: Group filtered ActivityItems by ActivityID.
		{{Key: "$group", Value: bson.M{
			"_id":           "$activityId",
			"activityItems": bson.M{"$push": "$$ROOT"}, // $$ROOT now has 'dates' as a correctly filtered array.
		}}},
		// Stage D: Lookup Activity details from 'activitys' collection.
		{{Key: "$lookup", Value: bson.M{
			"from":         "activitys",
			"localField":   "_id",
			"foreignField": "_id",
			"as":           "activityInfo",
		}}},
		// Stage E: Unwind the Activity details.
		{{Key: "$unwind", Value: "$activityInfo"}}, // Use {Key: "$unwind", Value: bson.M{"path": "$activityInfo", "preserveNullAndEmptyArrays": true}} if you want to keep activities that might not have items after filtering, or items whose activityId doesn't match an activity.

		// ท่านี้สั้น แต่อาจจะเข้าใจยากหน่อย performance น้อยกว่า เลยใช้ $project แทน
		// {{Key: "$replaceRoot", Value: bson.M{
		// 	"newRoot": bson.M{
		// 		"$mergeObjects": bson.A{"$activityInfo", bson.M{"activityItems": "$activityItems"}},
		// 	},
		// }}},

		{{Key: "$project", Value: bson.M{
			"id":              "$_id", // Exclude the default _id field
			"name":            "$activityInfo.name",
			"activityState":   "$activityInfo.activityState",
			"skill":           "$activityInfo.skill",
			"foodVotes":       "$activityInfo.foodVotes",
			"file":            "$activityInfo.file",
			"endDateEnroll":   "$activityInfo.endDateEnroll",
			"activityItems":   "$activityItems",
			"enrollmentCount": "$enrollmentCount",
		}}},
	}

}
