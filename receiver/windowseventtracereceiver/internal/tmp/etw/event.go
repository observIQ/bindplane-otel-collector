package etw

import "time"

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
