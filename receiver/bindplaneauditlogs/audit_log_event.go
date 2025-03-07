package bindplaneauditlogs

import "time"

// AuditEventAction indicates what the action was that acted on a resource
type AuditEventAction string

// TODO: Evaluate if this should actually be an enum, and evaluate the values for the enum.
var (
	// AuditEventActionCreated indicates an action that created the resource
	AuditEventActionCreated AuditEventAction = "Created"
	// AuditEventActionModified indicates an action that modified the resource
	AuditEventActionModified AuditEventAction = "Modified"
	// AuditEventActionDeleted indicates an action that deleted the resource
	AuditEventActionDeleted AuditEventAction = "Deleted"
	// AuditEventActionUnknown indicates an unknown action
	AuditEventActionUnknown AuditEventAction = "Unknown"
	// AuditEventActionPaused indicates an action that paused the rollout
	AuditEventActionStarted AuditEventAction = "Started"
	// AuditEventActionPaused indicates an action that paused the rollout
	AuditEventActionResumed AuditEventAction = "Resumed"
	// AuditEventActionPaused indicates an action that paused the rollout
	AuditEventActionPaused AuditEventAction = "Paused"
	// AuditEventActionPending indicates an action that created a pending rollout
	AuditEventActionPending AuditEventAction = "Pending"
)

// AuditLogEvent represents an event effecting a resource that is used to create an audit trail
type AuditLogEvent struct {
	// ID is a ULID uniquely identifying this event
	ID string `json:"id" db:"id"`
	// Timestamp is the time this event occurred
	Timestamp *time.Time `json:"timestamp" db:"created_at"`
	// ResourceName is the resource name + friendly name of the resource
	ResourceName string `json:"resourceName" db:"resource_name"`
	// Description is the friendly name of the resource
	Description string `json:"description" csv:"description" db:"description"`
	// ResourceKind is the resource that was modified
	ResourceKind Kind `json:"resourceKind" db:"resource_kind"`
	// Configuration is the name of the configuration affected. This may be nil if there is not associated configuration.
	Configuration string `json:"configuration,omitempty" db:"configuration_name"`
	// Action is the action that was taken on the resource
	Action AuditEventAction `json:"action" db:"action_type"`
	// User is the user that modified the resource
	User string `json:"user" db:"user_id"`
	// Account is the account that the event occurred on. May be nil in the case of single-account.
	Account string `json:"account,omitempty" db:"account"`
}

// Kind indicates the kind of resource, e.g. Configuration
type Kind string

// Kind values correspond to the kinds of resources currently supported by BindPlane
const (
	KindProfile                    Kind = "Profile"
	KindContext                    Kind = "Context"
	KindConfiguration              Kind = "Configuration"
	KindAgent                      Kind = "Agent"
	KindAgentType                  Kind = "AgentType"
	KindAgentVersion               Kind = "AgentVersion"
	KindSource                     Kind = "Source"
	KindProcessor                  Kind = "Processor"
	KindConnector                  Kind = "Connector"
	KindDestination                Kind = "Destination"
	KindExtension                  Kind = "Extension"
	KindSourceType                 Kind = "SourceType"
	KindProcessorType              Kind = "ProcessorType"
	KindConnectorType              Kind = "ConnectorType"
	KindDestinationType            Kind = "DestinationType"
	KindExtensionType              Kind = "ExtensionType"
	KindRecommendationType         Kind = "RecommendationType"
	KindUnknown                    Kind = "Unknown"
	KindRollout                    Kind = "Rollout"
	KindRolloutType                Kind = "RolloutType"
	KindOrganization               Kind = "Organization"
	KindAccount                    Kind = "Account"
	KindInvitation                 Kind = "Invitation"
	KindLogin                      Kind = "Login"
	KindUser                       Kind = "User"
	KindAccountOrganizationBinding Kind = "AccountOrganizationBinding"
	KindUserOrganizationBinding    Kind = "UserOrganizationBinding"
	KindUserAccountBinding         Kind = "UserAccountBinding"
	KindAuditEvent                 Kind = "AuditTrail"
	KindAvailableComponents        Kind = "AvailableComponents"
)
