package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetCheckinStatus(studentId, activityItemId string) ([]models.CheckinoutRecord, error) {
	uID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(activityItemId)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	filter := bson.M{"userId": uID, "activityItemId": aID}

	cursor, err := DB.CheckinCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถค้นหาข้อมูลเช็คชื่อได้")
	}
	defer cursor.Close(context.TODO())

	loc, _ := time.LoadLocation("Asia/Bangkok")

	type rec struct {
		Type      string    `bson:"type"`
		CheckedAt time.Time `bson:"checkedAt"`
	}

	var checkins, checkouts []time.Time
	for cursor.Next(context.TODO()) {
		var r rec
		if err := cursor.Decode(&r); err != nil {
			continue
		}
		t := r.CheckedAt.In(loc)
		switch r.Type {
		case "checkin":
			checkins = append(checkins, t)
		case "checkout":
			checkouts = append(checkouts, t)
		}
	}

	sort.Slice(checkins, func(i, j int) bool { return checkins[i].Before(checkins[j]) })
	sort.Slice(checkouts, func(i, j int) bool { return checkouts[i].Before(checkouts[j]) })

	used := make([]bool, len(checkouts))
	var results []models.CheckinoutRecord

	for _, ci := range checkins {
		ciCopy := ci // ต้อง copy เพื่ออ้าง pointer ได้ถูก
		var coPtr *time.Time
		for i, c := range checkouts {
			if !used[i] && c.After(ci) {
				cCopy := c
				coPtr = &cCopy
				used[i] = true
				break
			}
		}
		results = append(results, models.CheckinoutRecord{
			Checkin:  &ciCopy,
			Checkout: coPtr,
		})
	}

	return results, nil
}
