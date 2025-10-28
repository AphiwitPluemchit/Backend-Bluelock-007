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

// ส่งเมลเมื่อโปรแกรม “เสร็จสิ้น” ไปยังนิสิตที่ลงทะเบียนไว้ พร้อมสรุปชั่วโมงรวมต่อคน
func HandleNotifyProgramCompleted(
	sender MailSender,
	programResolver func(programID string) (*models.ProgramDto, error),
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p NotifyProgramCompletedPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}

		// โหลด Program (เพื่อดึง items, hours ฯลฯ)
		prog, err := programResolver(p.ProgramID)
		if err != nil || prog == nil {
			return fmt.Errorf("program not found: %s", p.ProgramID)
		}
		if len(prog.ProgramItems) == 0 {
			log.Printf("completed: no programItems for program %s", p.ProgramID)
			return nil
		}

		// เตรียม map สำหรับดู hour และ name ของแต่ละ item
		itemHour := make(map[primitive.ObjectID]int)
		itemName := make(map[primitive.ObjectID]string)
		itemIDs := make([]primitive.ObjectID, 0, len(prog.ProgramItems))
		for _, it := range prog.ProgramItems {
			itemHour[it.ID] = *it.Hour
			itemName[it.ID] = *it.Name
			itemIDs = append(itemIDs, it.ID)
		}

		// ดึง enrollments ทั้งหมดของ program (จากทุก item ของโปรแกรมนี้)
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
			// โหลดข้อมูลนิสิต
			var st models.Student
			if err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": en.StudentID}).Decode(&st); err != nil {
				continue
			}
			// บางที student ยังไม่มีในแอป ให้ข้าม
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

		base := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
		if base == "" {
			base = "http://localhost:9000"
		}
		detailURL := base + "/Student/Programs/" + p.ProgramID
		const emailDomain = "@go.buu.ac.th"

		for _, a := range byStudent {
			to := a.Code + emailDomain
			html, err := RenderCompletedEmailHTML(CompletedEmailData{
				StudentName: a.Name,
				Major:       a.Major,
				ProgramName: p.ProgramName,
				TotalHours:  a.TotalHours,
				Items:       a.Items,
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