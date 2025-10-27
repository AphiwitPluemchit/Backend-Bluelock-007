package email

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TypeNotifyProgramReminder = "email:notify-program-reminder"

type NotifyProgramReminderPayload struct {
	ProgramID    string `json:"programId"`
	ProgramName  string `json:"programName"`
	ProgramItemID string `json:"programItemId"` // เจาะจง ProgramItem ที่จะเตือน
}

func NewNotifyProgramReminderTask(programID, programName, programItemID string) (*asynq.Task, error) {
	b, err := json.Marshal(NotifyProgramReminderPayload{
		ProgramID:    programID,
		ProgramName:  programName,
		ProgramItemID: programItemID,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeNotifyProgramReminder, b), nil
}
