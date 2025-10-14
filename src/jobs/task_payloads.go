package jobs

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TypeCompleteProgram = "program:complete"

type ProgramPayload struct {
	ProgramID   string `json:"program_id"`
	ProgramName string `json:"program_name,omitempty"`
}

// NewCompleteProgramTaskWithName creates a complete-program task with id and optional name.
func NewCompleteProgramTaskWithName(programID, programName string) (*asynq.Task, error) {
	payload, err := json.Marshal(ProgramPayload{ProgramID: programID, ProgramName: programName})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCompleteProgram, payload), nil
}

// Deprecated: use NewCompleteProgramTaskWithName to provide a descriptive program name.
// Kept for backwards compatibility.
func NewCompleteProgramTask(programID string) (*asynq.Task, error) {
	return NewCompleteProgramTaskWithName(programID, "")
}

// jobs/task_payloads.go
const TypeCloseEnroll = "close:enroll"

// NewCloseEnrollTaskWithName creates a close-enroll task with id and optional name.
func NewCloseEnrollTaskWithName(programID, programName string) (*asynq.Task, error) {
	payload, err := json.Marshal(ProgramPayload{ProgramID: programID, ProgramName: programName})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCloseEnroll, payload), nil
}

// Deprecated: use NewCloseEnrollTaskWithName to provide a descriptive program name.
// Kept for backwards compatibility.
func NewCloseEnrollTask(programID string) (*asynq.Task, error) {
	return NewCloseEnrollTaskWithName(programID, "")
}
