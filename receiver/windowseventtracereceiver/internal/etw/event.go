package etw

import (
	"time"
)

type EventID uint16

type Event struct {
	Flags struct {
		// Use to flag event as being skippable for performance reason
		Skippable bool
	} `json:"-"`

	EventData map[string]interface{} `json:",omitempty"`
	UserData  map[string]interface{} `json:",omitempty"`
	System    struct {
		Channel     string
		Computer    string
		EventID     uint16
		EventType   string `json:",omitempty"`
		EventGuid   string `json:",omitempty"`
		Correlation struct {
			ActivityID        string
			RelatedActivityID string
		}
		Execution struct {
			ProcessID uint32
			ThreadID  uint32
		}
		Keywords struct {
			Value uint64
			Name  string
		}
		Level struct {
			Value uint8
			Name  string
		}
		Opcode struct {
			Value uint8
			Name  string
		}
		Task struct {
			Value uint8
			Name  string
		}
		Provider struct {
			Guid string
			Name string
		}
		TimeCreated struct {
			SystemTime time.Time
		}
	}
	ExtendedData []string `json:",omitempty"`
}

func NewEvent() (e *Event) {
	e = &Event{}
	e.EventData = make(map[string]interface{})
	e.UserData = make(map[string]interface{})
	e.ExtendedData = make([]string, 0)
	return e
}

func (e *Event) GetProperty(name string) (i interface{}, ok bool) {

	if e.EventData != nil {
		if i, ok = e.EventData[name]; ok {
			return
		}
	}

	if e.UserData != nil {
		if i, ok = e.UserData[name]; ok {
			return
		}
	}

	return
}

func (e *Event) GetPropertyString(name string) (string, bool) {
	if i, ok := e.GetProperty(name); ok {
		if s, ok := i.(string); ok {
			return s, ok
		}
	}
	return "", false
}
