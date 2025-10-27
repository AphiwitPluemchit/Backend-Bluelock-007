
package email

import (
	"bytes"
	_ "embed"
	"html/template"
	"time"

	"Backend-Bluelock-007/src/models"
)

//ส่งเมลเมื่อเปิดลงทะเบียน
type OpenEmailData struct {
	StudentName   string
	Major         string
	ProgramName   string
	EndDateEnroll string
	RegisterLink  string
	ProgramItems  []models.ProgramItemDto
}

//go:embed email_open_program.html
var openEmailHTML string

var openEmailTmpl = template.Must(
	template.New("open").
		Funcs(template.FuncMap{
			// ใช้ได้ในเทมเพลต: {{formatDate .Date}}
			"formatDate": func(s string) string {
				// s รูปแบบ "2006-01-02" => "02/01/2006"
				if t, err := time.Parse("2006-01-02", s); err == nil {
					return t.Format("02/01/2006")
				}
				return s
			},
		}).
		Parse(openEmailHTML),
)

func RenderOpenEmailHTML(data OpenEmailData) (string, error) {
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
