package email

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TypeNotifyProgramCompleted = "email:notify-program-completed"

type NotifyProgramCompletedPayload struct {
	ProgramID   string `json:"programId"`
	ProgramName string `json:"programName"`
}

func NewNotifyProgramCompletedTask(programID, programName string) (*asynq.Task, error) {
	p := NotifyProgramCompletedPayload{
		ProgramID:   programID,
		ProgramName: programName,
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeNotifyProgramCompleted, b), nil
}
