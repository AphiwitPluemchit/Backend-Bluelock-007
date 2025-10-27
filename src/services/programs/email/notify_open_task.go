package programs

import (
	"encoding/json"
	"strings"

	"github.com/hibiken/asynq"
)

const TypeNotifyOpenProgram = "programs:notify-open"

type NotifyOpenProgramPayload struct {
	ProgramID   string `json:"programId"`
	ProgramName string `json:"programName"`
}

func (p *NotifyOpenProgramPayload) Normalize() {
	p.ProgramID = strings.TrimSpace(p.ProgramID)
	p.ProgramName = strings.TrimSpace(p.ProgramName)
}

func NewNotifyOpenProgramTask(programID, programName string) (*asynq.Task, error) {
	payload := NotifyOpenProgramPayload{
		ProgramID:   programID,
		ProgramName: programName,
	}
	payload.Normalize()

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeNotifyOpenProgram, b), nil
}

func NotifyOpenTaskID(programID string) string {
	return "notify-open-" + strings.TrimSpace(programID)
}
