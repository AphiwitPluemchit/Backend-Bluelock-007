package jobs

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TypeCompleteProgram = "program:complete"

type ProgramPayload struct {
	ProgramID string `json:"program_id"`
}

func NewCompleteProgramTask(programID string) (*asynq.Task, error) {
	payload, err := json.Marshal(ProgramPayload{ProgramID: programID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCompleteProgram, payload), nil
}

// jobs/task_payloads.go
const TypeCloseEnroll = "close:enroll"

func NewCloseEnrollTask(programID string) (*asynq.Task, error) {
	payload, err := json.Marshal(ProgramPayload{ProgramID: programID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCloseEnroll, payload), nil
}
