package status

import (
	"fmt"

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

// Pump handles pumping a status report into the supplied pipeline.
func Pump(pipeline *message.Pipeline) error {
	report := getReport()

	reportMsg, err := message.NewMessage("statusReport", report)
	if err != nil {
		return fmt.Errorf("failed to create status report message: %w", err)
	}

	pipeline.Outbound() <- reportMsg
	return nil
}

// getReport returns a status report for the collector.
func getReport() Report {
	return Report{
		ComponentType: "bpagent",
		ComponentID:   "bpagent",
		Status:        ACTIVE,
	}
}
