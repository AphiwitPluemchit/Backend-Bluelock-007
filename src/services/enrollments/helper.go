package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"log"
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

	filter := bson.M{"studentId": uID, "activityItemId": aID}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cur, err := DB.CheckinCollection.Find(ctx, filter)
	log.Println("decode cur:", cur)

	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถค้นหาข้อมูลเช็คชื่อได้")
	}
	defer cur.Close(ctx)

	loc, _ := time.LoadLocation("Asia/Bangkok")

	type rec struct {
		Type      string    `bson:"type"`
		Timestamp time.Time `bson:"timestamp"`
	}

	var checkins, checkouts []time.Time
	var total int
	for cur.Next(ctx) {
		total++
		var r rec
		if err := cur.Decode(&r); err != nil {
			// ถ้า decode พัง จะรู้ทันทีว่า field/tag ไม่ตรง
			log.Printf("decode err: %v", err)
			continue
		}
		t := r.Timestamp.In(loc)
		switch r.Type {
		case "checkin":
			checkins = append(checkins, t)
		case "checkout":
			checkouts = append(checkouts, t)
		default:
			log.Printf("unknown type: %q", r.Type)
		}
	}
	if err := cur.Err(); err != nil {
		// error ระหว่าง iterate
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	// ถ้า total == 0 แปลว่า filter ไม่ตรง/คอลัมน์สะกดผิด
	log.Printf("[GetCheckinStatus] docs=%d, checkins=%d, checkouts=%d", total, len(checkins), len(checkouts))

	sort.Slice(checkins, func(i, j int) bool { return checkins[i].Before(checkins[j]) })
	sort.Slice(checkouts, func(i, j int) bool { return checkouts[i].Before(checkouts[j]) })

	used := make([]bool, len(checkouts))
	results := make([]models.CheckinoutRecord, 0, len(checkins))

	for _, ci := range checkins {
		ciCopy := ci
		var coPtr *time.Time
		for i, c := range checkouts {
			if !used[i] && (c.After(ci) || c.Equal(ci)) {
				cCopy := c
				coPtr = &cCopy
				used[i] = true
				break
			}
		}
		results = append(results, models.CheckinoutRecord{
			Checkin:  &ciCopy,
			Checkout: coPtr, // อาจเป็น nil ได้
		})
	}

	// ถ้าอยากให้กรณีมีแต่ checkout (ผิดลำดับ) ก็ยังได้ record ไว้ตรวจ
	// for i, c := range checkouts { if !used[i] { results = append(results, models.CheckinoutRecord{ Checkout: &c }) } }

	log.Printf("[GetCheckinStatus] results=%d", len(results))
	return results, nil
}
