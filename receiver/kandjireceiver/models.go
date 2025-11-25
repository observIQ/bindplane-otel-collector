package kandjireceiver

import "time"

type AuditEventsResponse struct {
	Results  []AuditEvent `json:"results"`
	Previous *string      `json:"previous"` // optional
	Next     *string      `json:"next"`     // optional
}

type AuditEvent struct {
	ID              string         `json:"id"`
	Action          string         `json:"action"`
	OccurredAt      string         `json:"occurred_at"`
	ActorID         string         `json:"actor_id"`
	ActorType       string         `json:"actor_type"`
	TargetID        string         `json:"target_id"`
	TargetType      string         `json:"target_type"`
	TargetComponent string         `json:"target_component"`
	NewState        map[string]any `json:"new_state"`
	Metadata        map[string]any `json:"metadata"`
}

type KandjiEndpoint string

type ParamType int

const (
	EPAuditEventsList KandjiEndpoint = "GET /audit/events"
)

const (
	ParamString ParamType = iota
	ParamInt
	ParamBool
	ParamTime
	ParamEnum
	ParamUUID
	ParamFloat
)

type ParamSpec struct {
	Name        string // query or path param name
	Type        ParamType
	Required    bool
	AllowedVals []string // optional: used when Type == ParamEnum
	Constraints *ParamConstraints
}

var CursorParam = ParamSpec{
	Name:     "cursor",
	Type:     ParamString,
	Required: false,
}

type ParamConstraints struct {
	// Numbers
	MinInt *int
	MaxInt *int

	// Strings
	MinLen *int
	MaxLen *int

	// Dates (RFC3339 string → parsed time)
	NotBefore       *time.Time     // cannot be earlier than this
	NotAfter        *time.Time     // cannot be later than this
	MaxAge          *time.Duration // e.g. 30 * 24 * time.Hour
	NotNewerThanNow bool           // disallow future timestamps
}

type EndpointSpec struct {
	Method             string
	Path               string
	Description        string
	Params             []ParamSpec
	ResponseType       any
	SupportsPagination bool
}

type scrapeStats struct {
	CallCount        int64
	ErrorCount       int64
	LatencyMs        float64
	Bytes            int64
	RecordCount      int64
	Pages            int64
	HTTPStatus       int
	Status           string
	ErrorReason      string
	ScrapeDurationMs float64
}
