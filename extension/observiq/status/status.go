package status

import (
	"github.com/observiq/observiq-collector/extension/observiq/message"
)

// Status is the status of the collector.
type Status int

const (
	DISABLED Status = 0
	ACTIVE   Status = 1
	ERROR    Status = 2
)

// Report is a status report.
type Report struct {
	ComponentType string `json:"componentType" mapstructure:"componentType"`
	ComponentID   string `json:"componentID" mapstructure:"componentID"`
	Status        Status `json:"status" mapstructure:"status"`
}

// ToMessage converts a report into a message.
func (r *Report) ToMessage() *message.Message {
	msg, _ := message.New(message.StatusReport, r)
	return msg
}

// Get returns the status of the collector.
func Get() (Report, error) {
	return Report{
		ComponentType: "bpagent",
		ComponentID:   "bpagent",
		Status:        ACTIVE,
	}, nil
}
