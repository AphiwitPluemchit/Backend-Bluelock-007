package jobs

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TypecompleteActivity = "activity:complete"

type ActivityPayload struct {
	ActivityID string `json:"activity_id"`
}

func NewcompleteActivityTask(activityID string) (*asynq.Task, error) {
	payload, err := json.Marshal(ActivityPayload{ActivityID: activityID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypecompleteActivity, payload), nil
}

// jobs/task_payloads.go
const TypeCloseEnroll = "close:enroll"

func NewCloseEnrollTask(activityID string) (*asynq.Task, error) {
	payload, err := json.Marshal(ActivityPayload{ActivityID: activityID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCloseEnroll, payload), nil
}
