package email

import (
	"log"
	"time"

	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"

	"github.com/hibiken/asynq"
)

// ScheduleReminderJobs
// ใช้หลังจากสร้างหรืออัปเดต program แล้ว
// จะสร้าง task แจ้งเตือนก่อนกิจกรรมเริ่ม 3 วัน (ตามเวลาเริ่มจริง)
func ScheduleReminderJobs(prog *models.ProgramDto) {
	if DB.AsynqClient == nil {
		log.Println("⚠️ Redis/Asynq not available → skip scheduling reminder jobs")
		return
	}

	for _, item := range prog.ProgramItems {
		if len(item.Dates) == 0 {
			continue
		}

		d0 := item.Dates[0]
		runAt, err := computeReminderTime(d0.Date, d0.Stime)
		if err != nil {
			log.Println("reminder: invalid date/time:", err)
			continue
		}

		// ถ้าเลยเวลาแล้ว ข้าม
		if runAt.Before(time.Now()) {
			continue
		}

		task, err := NewNotifyProgramReminderTask(
			prog.ID.Hex(),
			stringOrEmpty(prog.Name),
			item.ID.Hex(),
		)
		if err != nil {
			log.Println("reminder: create task failed:", err)
			continue
		}

		taskID := "reminder-3d-" + prog.ID.Hex() + "-" + item.ID.Hex()
		if _, err := DB.AsynqClient.Enqueue(
			task,
			asynq.ProcessAt(runAt),
			asynq.TaskID(taskID),
		); err != nil {
			log.Println("reminder: enqueue failed:", err)
		} else {
			log.Printf("✅ scheduled reminder: %s at %s", taskID, runAt.Format(time.RFC3339))
		}
	}
}

// 🕒 คำนวณเวลาแจ้งเตือนก่อน 3 วัน "เวลาเดียวกัน"
func computeReminderTime(dateStr, stime string) (time.Time, error) {
    if stime == "" { stime = "00:00" }
    loc, _ := time.LoadLocation("Asia/Bangkok")
    t, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+stime, loc)
    if err != nil { return time.Time{}, err }

    // ถ้าเป็น พ.ศ. ให้แปลงเป็น ค.ศ.
    if t.Year() >= 2400 {
        t = time.Date(t.Year()-543, t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
    }
    runAt := t.AddDate(0, 0, -3)
    log.Printf("📅 start=%s | reminder=%s", t.Format("2006-01-02 15:04"), runAt.Format("2006-01-02 15:04"))
    return runAt, nil
}

// ป้องกัน nil pointer จาก prog.Name
func stringOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
