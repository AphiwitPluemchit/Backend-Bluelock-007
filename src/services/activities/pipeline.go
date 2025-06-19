package activities

import (
	"Backend-Bluelock-007/src/models"
	"context"
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

	loc, _ := time.LoadLocation("Asia/Bangkok")
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
				{Key: "type", Value: bson.D{{Key: "$first", Value: "$type"}}},
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
			Key: "$match", Value: bson.D{
				{Key: "activityId", Value: activityID},
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
	cursor, err := activityItemCollection.Find(ctx, filter)
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
		for _, year := range generateStudentCodeFilter(studentYears) {
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
	countCursor, err := enrollmentCollection.Aggregate(ctx, countPipeline)
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

	cursor, err := enrollmentCollection.Aggregate(ctx, pipeline)
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
