package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetCheckinStatus(studentId, programItemId string) ([]models.CheckinoutRecord, error) {
	uID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(programItemId)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	filter := bson.M{"studentId": uID, "programItemId": aID}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	loc, locErr := time.LoadLocation("Asia/Bangkok")
	if locErr != nil {
		// fallback: UTC (แต่ควรแก้ให้โหลดได้ในโปรดักชัน)
		loc = time.FixedZone("UTC+7", 7*60*60)
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})
	cur, err := DB.CheckinCollection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถค้นหาข้อมูลเช็คชื่อได้")
	}
	defer cur.Close(ctx)

	type rec struct {
		Type      string    `bson:"type"`
		Timestamp time.Time `bson:"timestamp"`
	}

	var (
		checkins  []time.Time
		checkouts []time.Time
		total     int
	)

	for cur.Next(ctx) {
		total++
		var r rec
		if err := cur.Decode(&r); err != nil {
			log.Printf("[GetCheckinStatus] decode error: %v", err)
			continue
		}
		t := r.Timestamp.In(loc)
		switch r.Type {
		case "checkin":
			checkins = append(checkins, t)
		case "checkout":
			checkouts = append(checkouts, t)
		default:
			log.Printf("[GetCheckinStatus] unknown type: %q", r.Type)
		}
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	log.Printf("[GetCheckinStatus] docs=%d, checkins=%d, checkouts=%d", total, len(checkins), len(checkouts))

	// checkins/checkouts ถูก sort แล้วเพราะเรา sort ที่ query ไว้
	used := make([]bool, len(checkouts))
	results := make([]models.CheckinoutRecord, 0, max(len(checkins), len(checkouts)))

	// 1) จับคู่จากด้าน checkin ก่อน: earliest checkout >= checkin
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
			Checkout: coPtr, // อาจเป็น nil
		})
	}

	// 2) ถ้ามี checkout ที่ยังไม่ถูกใช้ (ไม่มี checkin นำหน้า) — เติมเป็นเรคคอร์ด checkout-only
	for i, c := range checkouts {
		if !used[i] {
			cCopy := c
			results = append(results, models.CheckinoutRecord{
				Checkin:  nil,
				Checkout: &cCopy,
			})
		}
	}

	// 3) เติมค่า Participation ต่อวันจาก ProgramItem.Dates
	var programItem models.ProgramItem
	if err := DB.ProgramItemCollection.FindOne(ctx, bson.M{"_id": aID}).Decode(&programItem); err == nil {
		// map วันที่ -> start time
		startByDate := make(map[string]time.Time, len(programItem.Dates))
		for _, d := range programItem.Dates {
			if d.Date == "" || d.Stime == "" {
				continue
			}
			if st, err := time.ParseInLocation("2006-01-02 15:04", d.Date+" "+d.Stime, loc); err == nil {
				startByDate[d.Date] = st
			}
		}

		presentDates := map[string]bool{}
		for i := range results {
			// หา key เป็นวันที่ของ checkin (ถ้ามี) ไม่งั้นใช้ checkout
			var dateKey string
			if results[i].Checkin != nil {
				dateKey = results[i].Checkin.In(loc).Format("2006-01-02")
			} else if results[i].Checkout != nil {
				dateKey = results[i].Checkout.In(loc).Format("2006-01-02")
			}

			participation := "ยังไม่เข้าร่วมกิจกรรม"
			hasIn := results[i].Checkin != nil
			hasOut := results[i].Checkout != nil

			switch {
			case hasIn && hasOut:
				if st, ok := startByDate[dateKey]; ok {
					// อนุโลม +/- 15 นาที
					early := st.Add(-15 * time.Minute)
					late := st.Add(15 * time.Minute)
					if (results[i].Checkin.Equal(early) || results[i].Checkin.After(early)) &&
						(results[i].Checkin.Before(late) || results[i].Checkin.Equal(late)) {
						participation = "เช็คอิน/เช็คเอาท์ตรงเวลา"
					} else {
						participation = "เช็คอิน/เช็คเอาท์ไม่ตรงเวลา"
					}
				} else {
					participation = "เช็คอิน/เช็คเอาท์ไม่เข้าเกณฑ์ (ไม่พบเวลาเริ่มกิจกรรมของวันนั้น)"
				}

			case hasIn && !hasOut:
				// ยังไม่เช็คเอาท์
				if st, ok := startByDate[dateKey]; ok && !results[i].Checkin.Before(st.Add(-15*time.Minute)) {
					participation = "เช็คอินแล้ว (รอเช็คเอาท์)"
				} else {
					participation = "เช็คอินแล้ว (เวลาไม่เข้าเกณฑ์)"
				}

			case !hasIn && hasOut:
				// เจอแต่ checkout
				participation = "เช็คเอาท์อย่างเดียว (ข้อมูลไม่ครบ)"
			}

			p := participation
			results[i].Participation = &p

			if dateKey != "" {
				presentDates[dateKey] = true
			}
		}

		// 4) เติมเรคคอร์ดเปล่าตามทุกวันที่กำหนดไว้ ถ้ายังไม่มีในผลลัพธ์
		for _, d := range programItem.Dates {
			if d.Date == "" {
				continue
			}
			if !presentDates[d.Date] {
				p := "ยังไม่เข้าร่วมกิจกรรม"
				results = append(results, models.CheckinoutRecord{
					Checkin:       nil,
					Checkout:      nil,
					Participation: &p,
				})
			}
		}
	} else {
		log.Printf("[GetCheckinStatus] programItem not found or decode error: %v", err)
	}

	log.Printf("[GetCheckinStatus] results=%d", len(results))
	return results, nil
}

// helper: max int
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
