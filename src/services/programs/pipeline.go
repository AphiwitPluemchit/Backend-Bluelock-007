package programs

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func getLightweightProgramsPipeline(
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
			{Key: "from", Value: "Program_Items"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "programId"},
			{Key: "as", Value: "programItems"},
		}}},
	}

	// set programItems into another field
	pipeline = append(pipeline, bson.D{{Key: "$set", Value: bson.D{
		{Key: "fullProgramItems", Value: "$programItems"},
	}},
	})

	// ✅ กรอง majors
	if len(majors) > 0 && majors[0] != "" {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{
			{Key: "programItems.majors", Value: bson.D{{Key: "$in", Value: majors}}},
		}}})
	}

	// ✅ กรอง studentYears
	if len(studentYears) > 0 && studentYears[0] != 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{
			{Key: "programItems.studentYears", Value: bson.D{{Key: "$in", Value: studentYears}}},
		}}})
	}

	// ✅ เงื่อนไขถ้าอยากเรียงตามวันจัดที่ใกล้สุด
	if sortField == "dates" {

		pipeline = append(pipeline,
			// Unwind programItems
			bson.D{{Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$programItems"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			}}},
			// Unwind programItems.dates
			bson.D{{Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$programItems.dates"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			}}})

		// Match เฉพาะวันในอนาคต ของ open close program
		if isSortNearest {
			pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{
				{Key: "programItems.dates.date", Value: bson.D{{Key: "$gte", Value: today}}},
			}}})
		}
		// Group เอาวันที่ใกล้ที่สุดต่อ program
		pipeline = append(pipeline, bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "nextDate", Value: bson.D{{Key: "$min", Value: "$programItems.dates.date"}}}, // nextDate คือ array ของ
			{Key: "name", Value: bson.D{{Key: "$first", Value: "$name"}}},
			{Key: "type", Value: bson.D{{Key: "$first", Value: "$type"}}},
			{Key: "programState", Value: bson.D{{Key: "$first", Value: "$programState"}}},
			{Key: "skill", Value: bson.D{{Key: "$first", Value: "$skill"}}},
			{Key: "file", Value: bson.D{{Key: "$first", Value: "$file"}}},
			{Key: "programItems", Value: bson.D{{Key: "$first", Value: "$fullProgramItems"}}},
		}}},

			// Sort nextDate
			bson.D{{Key: "$sort", Value: bson.D{{Key: "nextDate", Value: sortOrder}}}},
		)
	} else if sortField != "" {
		fmt.Println("sortField", sortField)
		fmt.Println("sortOrder", sortOrder)
		pipeline = append(pipeline,
			bson.D{{Key: "$sort", Value: bson.D{{Key: sortField, Value: sortOrder}}}},
		)
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

func GetOneProgramPipeline(programID primitive.ObjectID) mongo.Pipeline {
	return mongo.Pipeline{
		// 1️⃣ Match เฉพาะ Program ที่ต้องการ
		{{
			Key: "$match", Value: bson.D{
				{Key: "_id", Value: programID},
			},
		}},

		// 🔗 Lookup ProgramItems ที่เกี่ยวข้อง
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "Program_Items"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "programId"},
			{Key: "as", Value: "programItems"},
		}}},
	}
}

func GetProgramStatisticsPipeline(programID primitive.ObjectID) mongo.Pipeline {
	return mongo.Pipeline{
		// 1️⃣ Match เฉพาะ ProgramItems ที่ต้องการ
		{{
			Key: "$match", Value: bson.M{
				"programId": programID,
			},
		}},

		// 2️⃣ Lookup Enrollments จาก collection enrollments
		{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "Enrollments"},
				{Key: "localField", Value: "_id"},
				{Key: "foreignField", Value: "programItemId"},
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
				{Key: "from", Value: "Students"},
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

		// 6️⃣ Group ตาม ProgramItem และ Major
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: bson.D{
					{Key: "programItemId", Value: "$_id"},
					{Key: "majorName", Value: "$student.major"},
				}},
				{Key: "programItemName", Value: bson.D{{Key: "$first", Value: "$name"}}},
				{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
				{Key: "maxParticipants", Value: bson.D{{Key: "$first", Value: "$maxParticipants"}}},
			},
		}},

		// 9️⃣ Group ProgramItemSums
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$_id.programItemId"},
				{Key: "programItemName", Value: bson.D{{Key: "$first", Value: "$programItemName"}}},
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
				{Key: "programItemSums", Value: bson.D{{Key: "$push", Value: bson.D{
					{Key: "programItemName", Value: "$programItemName"},
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
				{Key: "programItemSums", Value: 1},
			},
		}},
	}
}
func GetProgramItemIDsByProgramID(ctx context.Context, programID primitive.ObjectID) ([]primitive.ObjectID, error) {
	var programItems []models.ProgramItem
	filter := bson.M{"programId": programID}
	cursor, err := DB.ProgramItemCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &programItems); err != nil {
		return nil, err
	}

	var programItemIDs []primitive.ObjectID
	for _, item := range programItems {
		programItemIDs = append(programItemIDs, item.ID)
	}

	return programItemIDs, nil
}

func GetProgramsPipeline(filter bson.M, sortField string, sortOrder int, skip int64, limit int64, majors []string, studentYears []int) mongo.Pipeline {
	pipeline := mongo.Pipeline{
		// 🔍 Match เฉพาะ Program ที่ต้องการ
		{{Key: "$match", Value: filter}},

		// 🔗 Lookup ProgramItems ที่เกี่ยวข้อง
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "Program_Items"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "programId"},
			{Key: "as", Value: "programItems"},
		}}},

		// 🔥 Unwind ProgramItems เพื่อให้สามารถกรองได้
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$programItems"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},

		// 3️⃣ Lookup EnrollmentCount แทนที่จะดึงทั้ง array
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "Enrollments"},
			{Key: "let", Value: bson.D{{Key: "itemId", Value: "$programItems._id"}}}, // let คือ การประกาศตัวแปรใน pipeline
			{Key: "pipeline", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{
					{Key: "$expr", Value: bson.D{ // ใช้ $expr เพื่อใช้ตัวแปรที่ประกาศใน let
						{Key: "$eq", Value: bson.A{"$programItemId", "$$itemId"}},
					}},
				}}},
				bson.D{{Key: "$count", Value: "count"}},
			}},
			{Key: "as", Value: "programItems.enrollmentCountData"},
		}}},

		// 4️⃣ Add enrollmentCount field จาก enrollmentCountData
		{{Key: "$addFields", Value: bson.D{
			{Key: "programItems.enrollmentCount", Value: bson.D{
				{Key: "$ifNull", Value: bson.A{bson.D{
					{Key: "$arrayElemAt", Value: bson.A{"$programItems.enrollmentCountData.count", 0}},
				}, 0}},
			}},
		}}},
	}

	// ✅ กรองเฉพาะ Major ที่ต้องการ **ถ้ามีค่า major**
	if len(majors) > 0 && majors[0] != "" {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "programItems.majors", Value: bson.D{{Key: "$in", Value: majors}}},
			}},
		})
	} else {
		fmt.Println("Skipping majorName filtering")
	}

	// ✅ กรองเฉพาะ StudentYears ที่ต้องการ **ถ้ามีค่า studentYears**
	if len(studentYears) > 0 {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "programItems.studentYears", Value: bson.D{{Key: "$in", Value: studentYears}}},
			}},
		})
	}

	// ✅ Group ProgramItems กลับเข้าไปใน Program
	pipeline = append(pipeline, bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "name", Value: bson.D{{Key: "$first", Value: "$name"}}},
			{Key: "type", Value: bson.D{{Key: "$first", Value: "$type"}}},
			{Key: "programState", Value: bson.D{{Key: "$first", Value: "$programState"}}},
			{Key: "skill", Value: bson.D{{Key: "$first", Value: "$skill"}}},
			{Key: "file", Value: bson.D{{Key: "$first", Value: "$file"}}},
			{Key: "programItems", Value: bson.D{{Key: "$push", Value: "$programItems"}}}, // เก็บ ProgramItems เป็น Array
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

func GetAllProgramCalendarPipeline(startDateStr string, endDateStr string) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"dates": bson.M{
				"$elemMatch": bson.M{
					"date": bson.M{
						// ดึงเพิ่ม 6 วันข้างหน้าและ 6 วันข้างหลัง
						"$gte": startDateStr + "-06",
						"$lte": endDateStr + "+06",
					},
				},
			},
		}}},

		// lookup enrollment
		{{Key: "$lookup", Value: bson.M{
			"from": "Enrollments",
			"let":  bson.M{"programItemId": "$_id"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{
						"$eq": bson.A{"$programItemId", "$$programItemId"},
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

		// Stage C: Group filtered ProgramItems by ProgramID.
		{{Key: "$group", Value: bson.M{
			"_id":          "$programId",
			"programItems": bson.M{"$push": "$$ROOT"}, // $$ROOT now has 'dates' as a correctly filtered array.
		}}},
		// Stage D: Lookup Program details from 'Programs' collection.
		{{Key: "$lookup", Value: bson.M{
			"from":         "Programs",
			"localField":   "_id",
			"foreignField": "_id",
			"as":           "programInfo",
		}}},
		// Stage E: Unwind the Program details.
		{{Key: "$unwind", Value: "$programInfo"}}, // Use {Key: "$unwind", Value: bson.M{"path": "$programInfo", "preserveNullAndEmptyArrays": true}} if you want to keep programs that might not have items after filtering, or items whose programId doesn't match an program.

		// ท่านี้สั้น แต่อาจจะเข้าใจยากหน่อย performance น้อยกว่า เลยใช้ $project แทน
		// {{Key: "$replaceRoot", Value: bson.M{
		// 	"newRoot": bson.M{
		// 		"$mergeObjects": bson.A{"$programInfo", bson.M{"programItems": "$programItems"}},
		// 	},
		// }}},

		{{Key: "$project", Value: bson.M{
			"id":              "$_id", // Exclude the default _id field
			"name":            "$programInfo.name",
			"type":            "$programInfo.type",
			"programState":    "$programInfo.programState",
			"skill":           "$programInfo.skill",
			"foodVotes":       "$programInfo.foodVotes",
			"file":            "$programInfo.file",
			"endDateEnroll":   "$programInfo.endDateEnroll",
			"programItems":    "$programItems",
			"enrollmentCount": "$enrollmentCount",
		}}},
	}

}
