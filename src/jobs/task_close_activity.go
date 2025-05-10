package jobs

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TypeCloseActivity = "activity:close"

type CloseActivityPayload struct {
	ActivityID string `json:"activity_id"`
}

func NewCloseActivityTask(activityID string) (*asynq.Task, error) {
	payload, err := json.Marshal(CloseActivityPayload{ActivityID: activityID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeCloseActivity, payload), nil
}
