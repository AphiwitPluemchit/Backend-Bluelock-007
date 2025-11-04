package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"

	"Backend-Bluelock-007/src/models"
)

var _ embed.FS 

type OpenEmailData struct {
	StudentName     string
	Major           string
	ProgramName     string
	EndDateEnroll   string
	RegisterLink    string
	ProgramItems    []models.ProgramItemDto
	Skill           string
	TotalHours      int
	MaxParticipants int
	Location        string
	Description     string
	Dates           []models.Dates
	StartTime       string
	EndTime         string
}

//go:embed email_open_program.html
var openEmailHTML string

var openEmailTmpl = template.Must(
	template.New("open").
		Funcs(template.FuncMap{
			"formatDate": func(s string) string {
				if t, err := time.Parse("2006-01-02", s); err == nil {
					return t.Format("02/01/2006")
				}
				return s
			},
			"formatDateThai": func(s string) string {
				loc, _ := time.LoadLocation("Asia/Bangkok")
				t, err := time.ParseInLocation("2006-01-02", s, loc)
				if err != nil {
					return s
				}
				months := []string{"", "มกราคม","กุมภาพันธ์","มีนาคม","เมษายน","พฤษภาคม","มิถุนายน","กรกฎาคม","สิงหาคม","กันยายน","ตุลาคม","พฤศจิกายน","ธันวาคม"}
				return fmt.Sprintf("%d %s %d", t.Day(), months[int(t.Month())], t.Year()+543)
			},
		}).
		Parse(openEmailHTML),
)

func RenderOpenEmailHTML(data OpenEmailData) (string, error) {
	// 🟩 แปลง skill จาก "Soft"/"Hard" เป็นข้อความภาษาไทย
	switch strings.ToLower(data.Skill) {
	case "soft":
		data.Skill = "ชั่วโมงเตรียมความพร้อม"
	case "hard":
		data.Skill = "ชั่วโมงทักษะทางวิชาการ"
	default:
		data.Skill = "ไม่ระบุประเภททักษะ"
	}

	var buf bytes.Buffer
	if err := openEmailTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

//ส่งเมลก่อน 3 วันก่อนเริ่มกิจกรรม
type ReminderEmailData struct {
	StudentName string
	Major       string
	ProgramName string

	FirstDate  string
	FirstStime string
	FirstEtime string

	DetailLink  string
	ProgramItem models.ProgramItemDto
}

//go:embed email_reminder_program.html
var reminderEmailHTML string

func RenderReminderEmailHTML(data ReminderEmailData) (string, error) {
	tmpl, err := template.New("reminder").Parse(reminderEmailHTML)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// --- Completed Email ---

type CompletedItem struct {
	Name string
	Hour int
}

type CompletedEmailData struct {
	StudentName string
	Major       string
	ProgramName string
	TotalHours  int
	Items       []CompletedItem
	DetailLink  string
}

//go:embed email_completed_program.html
var completedEmailHTML string

func RenderCompletedEmailHTML(data CompletedEmailData) (string, error) {
	tmpl, err := template.New("completed").Parse(completedEmailHTML)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
