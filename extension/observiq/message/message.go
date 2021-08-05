package message

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

const (
	StatusReport = "statusReport"
	TaskRequest  = "task"
	TaskResponse = "taskResponse"
)

// Message represents a message sent or received from observiq cloud.
type Message struct {
	Type    string                 `json:"type" mapstructure:"type"`
	Content map[string]interface{} `json:"content" mapstructure:"content"`
}

// New will create a new message with the supplied type and content.
func New(msgType string, msgContent interface{}) (*Message, error) {
	content := make(map[string]interface{})
	if err := mapstructure.Decode(msgContent, &content); err != nil {
		return nil, fmt.Errorf("unable to encode content: %s", err)
	}
	return &Message{Type: msgType, Content: content}, nil
}
