// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package macosendpointsecurityreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosendpointsecurityreceiver"

import (
	"encoding/json"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

// EventParser defines the interface for parsing Endpoint Security events
type EventParser interface {
	ParseEvent(data []byte) (EndpointSecurityEvent, error)
	GetEventType() EventType
}

// EndpointSecurityEvent is the base interface for all Endpoint Security events
type EndpointSecurityEvent interface {
	GetEventType() EventType
	GetTimestamp() time.Time
	GetProcess() *ProcessInfo
	ToLogRecord(logRecord plog.LogRecord)
}

// ProcessInfo represents process information in an event
type ProcessInfo struct {
	ExecutablePath string                 `json:"executable.path,omitempty"`
	PID            int32                  `json:"audit_token.pid,omitempty"`
	AuditToken     *AuditToken            `json:"audit_token,omitempty"`
	Attributes     map[string]interface{} `json:"-"`
}

// AuditToken represents audit token information
type AuditToken struct {
	PID    int32 `json:"pid,omitempty"`
	UID    int32 `json:"uid,omitempty"`
	GID    int32 `json:"gid,omitempty"`
	EUID   int32 `json:"euid,omitempty"`
	EGID   int32 `json:"egid,omitempty"`
	RUID   int32 `json:"ruid,omitempty"`
	RGID   int32 `json:"rgid,omitempty"`
}

// BaseEvent contains common fields for all events
type BaseEvent struct {
	EventType     EventType `json:"-"`
	Timestamp     time.Time `json:"-"`
	Process       *ProcessInfo
	SchemaVersion string `json:"schema_version,omitempty"`
	Version       int    `json:"version,omitempty"`
	RawJSON       string `json:"-"`
}

// GetEventType returns the event type
func (b *BaseEvent) GetEventType() EventType {
	return b.EventType
}

// GetTimestamp returns the event timestamp
func (b *BaseEvent) GetTimestamp() time.Time {
	return b.Timestamp
}

// GetProcess returns the process information
func (b *BaseEvent) GetProcess() *ProcessInfo {
	return b.Process
}

// ToLogRecord converts the base event to an OpenTelemetry log record
func (b *BaseEvent) ToLogRecord(logRecord plog.LogRecord) {
	logRecord.Body().SetStr(b.RawJSON)
	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(b.Timestamp))
	logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	
	// Set severity based on event type (security events are generally informational)
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
	logRecord.SetSeverityText("Info")
}

// GenericEventParser is a fallback parser for events without specific parsers
type GenericEventParser struct {
	eventType EventType
}

// NewGenericEventParser creates a new generic event parser
func NewGenericEventParser(eventType EventType) *GenericEventParser {
	return &GenericEventParser{eventType: eventType}
}

// GetEventType returns the event type this parser handles
func (g *GenericEventParser) GetEventType() EventType {
	return g.eventType
}

// ParseEvent parses a generic event from JSON
func (g *GenericEventParser) ParseEvent(data []byte) (EndpointSecurityEvent, error) {
	var eventData map[string]interface{}
	if err := json.Unmarshal(data, &eventData); err != nil {
		return nil, err
	}

	baseEvent := &BaseEvent{
		EventType: g.eventType,
		RawJSON:   string(data),
		Process:   &ProcessInfo{},
	}

	// Extract timestamp if available
	if ts, ok := eventData["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			baseEvent.Timestamp = t
		} else {
			baseEvent.Timestamp = time.Now()
		}
	} else {
		baseEvent.Timestamp = time.Now()
	}

	// Extract process information if available
	if processData, ok := eventData["process"].(map[string]interface{}); ok {
		if execPath, ok := processData["executable"].(map[string]interface{}); ok {
			if path, ok := execPath["path"].(string); ok {
				baseEvent.Process.ExecutablePath = path
			}
		}
		if auditToken, ok := processData["audit_token"].(map[string]interface{}); ok {
			baseEvent.Process.AuditToken = &AuditToken{}
			if pid, ok := auditToken["pid"].(float64); ok {
				baseEvent.Process.AuditToken.PID = int32(pid)
				baseEvent.Process.PID = int32(pid)
			}
		}
	}

	// Extract schema version and version
	if sv, ok := eventData["schema_version"].(string); ok {
		baseEvent.SchemaVersion = sv
	}
	if v, ok := eventData["version"].(float64); ok {
		baseEvent.Version = int(v)
	}

	return baseEvent, nil
}

// EventParserRegistry maps event types to their parsers
type EventParserRegistry map[EventType]EventParser

// NewEventParserRegistry creates a registry with generic parsers for all event types
func NewEventParserRegistry() EventParserRegistry {
	registry := make(EventParserRegistry)
	
	// Create generic parsers for all known event types
	for eventType := range ValidEventTypes {
		registry[eventType] = NewGenericEventParser(eventType)
	}
	
	return registry
}

// GetParser returns the parser for a given event type, or a generic parser if not found
func (r EventParserRegistry) GetParser(eventType EventType) EventParser {
	if parser, ok := r[eventType]; ok {
		return parser
	}
	return NewGenericEventParser(eventType)
}
