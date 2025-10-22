// file: src/services/email/notify_reminder_handler.go
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
	"time"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ✨ เพิ่มพารามิเตอร์ programResolver เข้ามา เพื่อตัดวงจร import cycle
func HandleProgramReminder(sender MailSender,
	programResolver func(programID string) (*models.ProgramDto, error),
) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p NotifyProgramReminderPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}

		// ✅ โหลด Program ผ่าน resolver ที่ฉีดเข้ามา
		prog, err := programResolver(p.ProgramID)
		if err != nil || prog == nil {
			return fmt.Errorf("program not found: %s", p.ProgramID)
		}

		// หา ProgramItem ตาม payload
		var item *models.ProgramItemDto
		pid, _ := primitive.ObjectIDFromHex(p.ProgramItemID)
		for i := range prog.ProgramItems {
			if prog.ProgramItems[i].ID == pid {
				item = &prog.ProgramItems[i]
				break
			}
		}
		if item == nil {
			return fmt.Errorf("program item not found: %s", p.ProgramItemID)
		}
		if len(item.Dates) == 0 {
			log.Println("reminder: item has no dates, skip")
			return nil
		}

		first := item.Dates[0]
		base := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
		if base == "" {
			base = "http://localhost:9000"
		}
		detailURL := base + "/Student/Programs/" + p.ProgramID

		cur, err := DB.EnrollmentCollection.Find(ctx, bson.M{"programItemId": item.ID})
		if err != nil {
			return err
		}
		defer cur.Close(ctx)

		const emailDomain = "@go.buu.ac.th"

		for cur.Next(ctx) {
			var en models.Enrollment
			if err := cur.Decode(&en); err != nil {
				continue
			}
			var st models.Student
			if err := DB.StudentCollection.FindOne(ctx, bson.M{"_id": en.StudentID}).Decode(&st); err != nil {
				continue
			}

			to := st.Code + emailDomain
			html, err := RenderReminderEmailHTML(ReminderEmailData{
				StudentName: st.Name,
				Major:       st.Major,
				ProgramName: p.ProgramName,
				FirstDate:   first.Date,
				FirstStime:  first.Stime,
				FirstEtime:  first.Etime,
				DetailLink:  detailURL,
				ProgramItem: *item,
			})
			if err != nil {
				log.Printf("reminder: render failed for %s: %v", to, err)
				continue
			}
			subject := "แจ้งเตือนก่อนกิจกรรม 3 วัน: " + p.ProgramName
			if err := sender.Send(to, subject, html); err != nil {
				log.Printf("reminder: send failed to %s: %v", to, err)
			}
		}

		log.Printf("reminder: sent for program=%s item=%s date=%s %s",
		 p.ProgramID, p.ProgramItemID, first.Date, time.Now().Format(time.RFC3339))
		return nil
	}
}
