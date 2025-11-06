// file: src/services/email/notify_completed_handler.go
package email

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func HandleNotifyProgramCompleted(
	sender MailSender,
	programResolver func(programID string) (*models.ProgramDto, error),
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p NotifyProgramCompletedPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}

		// โหลด Program
		prog, err := programResolver(p.ProgramID)
		if err != nil || prog == nil {
			return fmt.Errorf("program not found: %s", p.ProgramID)
		}
		if len(prog.ProgramItems) == 0 {
			log.Printf("completed: no programItems for program %s", p.ProgramID)
			return nil
		}

		// -------- รวมข้อมูลระดับโปรแกรม (เหมือน open/reminder) --------
		// Description: เอาจาก item แรก ถ้าไม่มีเป็น "-"
		desc := "-"
		if prog.ProgramItems[0].Description != nil && *prog.ProgramItems[0].Description != "" {
			desc = *prog.ProgramItems[0].Description
		}

		// Location: รวมห้อง/สถานที่จากทุก item (unique)
		roomSet := map[string]struct{}{}
		for _, it := range prog.ProgramItems {
			if it.Rooms == nil {
				continue
			}
			for _, r := range *it.Rooms {
				r = strings.TrimSpace(r)
				if r != "" {
					roomSet[r] = struct{}{}
				}
			}
		}
		locations := make([]string, 0, len(roomSet))
		for k := range roomSet {
			locations = append(locations, k)
		}
		location := strings.Join(locations, ", ")
		if location == "" {
			location = "-"
		}

		// Dates ทั้งโปรแกรม + หา earliest start / latest end
		var allDates []models.Dates
		minStart := "" // HH:MM
		maxEnd := ""   // HH:MM
		for _, it := range prog.ProgramItems {
			for _, d := range it.Dates {
				allDates = append(allDates, d)
				if d.Stime != "" && (minStart == "" || d.Stime < minStart) {
					minStart = d.Stime
				}
				if d.Etime != "" && (maxEnd == "" || d.Etime > maxEnd) {
					maxEnd = d.Etime
				}
			}
		}

		// เตรียมข้อมูล mapping hour/name ของแต่ละ item ไว้สรุปผลต่อคน
		itemHour := make(map[primitive.ObjectID]int)
		itemName := make(map[primitive.ObjectID]string)
		itemIDs := make([]primitive.ObjectID, 0, len(prog.ProgramItems))
		for _, it := range prog.ProgramItems {
			if it.Hour != nil {
				itemHour[it.ID] = *it.Hour
			} else {
				itemHour[it.ID] = 0
			}
			if it.Name != nil {
				itemName[it.ID] = *it.Name
			} else {
				itemName[it.ID] = "-"
			}
			itemIDs = append(itemIDs, it.ID)
		}

		// ดึง enrollments ทั้งหมดของ program
		cur, err := DB.EnrollmentCollection.Find(ctx, bson.M{
			"programItemId": bson.M{"$in": itemIDs},
		})
		if err != nil {
			return err
		}
		defer cur.Close(ctx)

		// สรุปชั่วโมงรายนิสิต
		type agg struct {
			Items      []CompletedItem
			TotalHours int
			Major      string
			Name       string
			Code       string
		}
		byStudent := map[primitive.ObjectID]*agg{}

		for cur.Next(ctx) {
			var en models.Enrollment
			if err := cur.Decode(&en); err != nil {
				continue
			}
			var st models.Student
			if err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": en.StudentID}).Decode(&st); err != nil {
				continue
			}
			if st.ID.IsZero() || st.Code == "" {
				continue
			}

			a, ok := byStudent[st.ID]
			if !ok {
				a = &agg{Major: st.Major, Name: st.Name, Code: st.Code}
				byStudent[st.ID] = a
			}

			hr := itemHour[en.ProgramItemID]
			nm := itemName[en.ProgramItemID]
			if hr < 0 {
				hr = 0
			}
			a.Items = append(a.Items, CompletedItem{Name: nm, Hour: hr})
			a.TotalHours += hr
		}

		if len(byStudent) == 0 {
			log.Printf("completed: no enrollments to notify for program %s", p.ProgramID)
			return nil
		}

		// ลิงก์ไปหน้ารายละเอียด/ประวัติ
		base := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
		if base == "" {
			base = "http://localhost:9000"
		}
		detailURL := base + "/Student/Programs/" + p.ProgramID
		const emailDomain = "@go.buu.ac.th"

		// ส่งเมลพร้อมข้อมูลครบชุด
		for _, a := range byStudent {
			to := a.Code + emailDomain
			html, err := RenderCompletedEmailHTML(CompletedEmailData{
				StudentName: a.Name,
				Major:       a.Major,
				ProgramName: p.ProgramName,

				// ฟิลด์สรุประดับโปรแกรม (ให้เทมเพลตโชว์ header แบบเดียวกับ open/reminder)
				Skill:         prog.Skill,          // จะถูกแปลงเป็นไทยใน RenderCompletedEmailHTML แล้ว
				Description:   desc,                // ถ้าไม่มีเป็น "-"
				Location:      location,            // รวมห้องทั้งหมดหรือ "-"
				EndDateEnroll: prog.EndDateEnroll,  // ใช้แสดงกำหนดเส้นตายได้ตามเทมเพลต
				ProgramItems:  prog.ProgramItems,   // เผื่อเทมเพลตต้องวนรายการ
				Dates:         allDates,            // ใช้ formatDateThai ในเทมเพลต
				StartTime:     minStart,
				EndTime:       maxEnd,

				TotalHours: a.TotalHours,		
				DetailLink:  detailURL,
				
			})
			if err != nil {
				log.Printf("completed: render failed for %s: %v", to, err)
				continue
			}

			subject := "ได้รับชั่วโมงอบรมแล้ว: " + p.ProgramName
			if err := sender.Send(to, subject, html); err != nil {
				log.Printf("completed: send failed to %s: %v", to, err)
			}
		}

		log.Printf("completed: notify done for program=%s", p.ProgramID)
		return nil
	}
}
