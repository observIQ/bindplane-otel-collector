package task

import (
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/observiq/observiq-collector/extension/observiq/message"
)

// Status is the status of a task.
type Status int

const (
	RUNNING   Status = 1
	SUCCESS   Status = 2
	FAILURE   Status = 3
	EXCEPTION Status = 4
)

// Task is a request to execute a specific task.
type Task struct {
	Type       string                 `json:"type" mapstructure:"type"`
	ID         string                 `json:"id" mapstructure:"id"`
	Parameters map[string]interface{} `json:"parameters" mapstructure:"parameters"`
}

// Response is the response to an executed task.
type Response struct {
	Type    string                 `json:"type" mapstructure:"type"`
	ID      string                 `json:"id" mapstructure:"id"`
	Status  Status                 `json:"status" mapstructure:"status"`
	Message string                 `json:"message" mapstructure:"message"`
	Details map[string]interface{} `json:"details" mapstructure:"details"`
}

// ToMessage converts the task response into a message.
func (r *Response) ToMessage() *message.Message {
	msg, _ := message.New(message.TaskResponse, r)
	return msg
}

// FromMessage creates a new task from the supplied message.
func FromMessage(msg *message.Message) (*Task, error) {
	if msg.Type != message.TaskRequest {
		return nil, fmt.Errorf("invalid message type: %s", msg.Type)
	}

	var task Task
	err := mapstructure.Decode(msg.Content, &task)
	if err != nil {
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	return &task, nil
}

// Execute will execute the supplied task
func Execute(task *Task) (*Response, error) {
	return nil, errors.New("unimplemented task type")
}
