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

// ใช้ชื่อ TaskID แบบคงที่ต่อโปรแกรม เพื่อ “ลบของเดิมก่อน enqueue”
func NotifyCompletedTaskID(programID string) string {
	return "notify-completed-" + programID
}

func NewNotifyProgramCompletedTask(programID, programName string) (*asynq.Task, error) {
	p := NotifyProgramCompletedPayload{
		ProgramID:   programID,
		ProgramName: programName,
	}
	return asynq.NewTask(TypeNotifyProgramCompleted, mustJSON(p)), nil
}

// helper เล็ก ๆ
func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
