// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package macosendpointsecurityreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosendpointsecurityreceiver"

import (
	"fmt"
)

// EventType represents a valid Endpoint Security event type
type EventType string

// All valid event types as constants
const (
	EventTypeAccess                    EventType = "access"
	EventTypeAuthentication            EventType = "authentication"
	EventTypeAuthorizationJudgement    EventType = "authorization_judgement"
	EventTypeAuthorizationPetition     EventType = "authorization_petition"
	EventTypeBtmLaunchItemAdd          EventType = "btm_launch_item_add"
	EventTypeBtmLaunchItemRemove       EventType = "btm_launch_item_remove"
	EventTypeChdir                     EventType = "chdir"
	EventTypeChroot                    EventType = "chroot"
	EventTypeClone                     EventType = "clone"
	EventTypeClose                     EventType = "close"
	EventTypeCopyfile                  EventType = "copyfile"
	EventTypeCreate                    EventType = "create"
	EventTypeCsInvalidated             EventType = "cs_invalidated"
	EventTypeDeleteextattr             EventType = "deleteextattr"
	EventTypeDup                       EventType = "dup"
	EventTypeExchangedata              EventType = "exchangedata"
	EventTypeExec                      EventType = "exec"
	EventTypeExit                      EventType = "exit"
	EventTypeFcntl                     EventType = "fcntl"
	EventTypeFileProviderMaterialize   EventType = "file_provider_materialize"
	EventTypeFileProviderUpdate        EventType = "file_provider_update"
	EventTypeFork                      EventType = "fork"
	EventTypeFsgetpath                 EventType = "fsgetpath"
	EventTypeGatekeeperUserOverride    EventType = "gatekeeper_user_override"
	EventTypeGetTask                   EventType = "get_task"
	EventTypeGetTaskInspect            EventType = "get_task_inspect"
	EventTypeGetTaskName               EventType = "get_task_name"
	EventTypeGetTaskRead               EventType = "get_task_read"
	EventTypeGetattrlist               EventType = "getattrlist"
	EventTypeGetextattr                EventType = "getextattr"
	EventTypeIokitOpen                 EventType = "iokit_open"
	EventTypeKextload                  EventType = "kextload"
	EventTypeKextunload                EventType = "kextunload"
	EventTypeLink                      EventType = "link"
	EventTypeListextattr               EventType = "listextattr"
	EventTypeLoginLogin                EventType = "login_login"
	EventTypeLoginLogout               EventType = "login_logout"
	EventTypeLookup                    EventType = "lookup"
	EventTypeLwSessionLock             EventType = "lw_session_lock"
	EventTypeLwSessionLogin             EventType = "lw_session_login"
	EventTypeLwSessionLogout            EventType = "lw_session_logout"
	EventTypeLwSessionUnlock            EventType = "lw_session_unlock"
	EventTypeMmap                      EventType = "mmap"
	EventTypeMount                     EventType = "mount"
	EventTypeMprotect                  EventType = "mprotect"
	EventTypeOdAttributeSet            EventType = "od_attribute_set"
	EventTypeOdAttributeValueAdd       EventType = "od_attribute_value_add"
	EventTypeOdAttributeValueRemove   EventType = "od_attribute_value_remove"
	EventTypeOdCreateGroup             EventType = "od_create_group"
	EventTypeOdCreateUser              EventType = "od_create_user"
	EventTypeOdDeleteGroup             EventType = "od_delete_group"
	EventTypeOdDeleteUser              EventType = "od_delete_user"
	EventTypeOdDisableUser             EventType = "od_disable_user"
	EventTypeOdEnableUser              EventType = "od_enable_user"
	EventTypeOdGroupAdd                EventType = "od_group_add"
	EventTypeOdGroupRemove             EventType = "od_group_remove"
	EventTypeOdGroupSet                EventType = "od_group_set"
	EventTypeOdModifyPassword          EventType = "od_modify_password"
	EventTypeOpen                      EventType = "open"
	EventTypeOpensshLogin              EventType = "openssh_login"
	EventTypeOpensshLogout             EventType = "openssh_logout"
	EventTypeProcCheck                 EventType = "proc_check"
	EventTypeProcSuspendResume         EventType = "proc_suspend_resume"
	EventTypeProfileAdd                EventType = "profile_add"
	EventTypeProfileRemove             EventType = "profile_remove"
	EventTypePtyClose                  EventType = "pty_close"
	EventTypePtyGrant                  EventType = "pty_grant"
	EventTypeReaddir                   EventType = "readdir"
	EventTypeReadlink                  EventType = "readlink"
	EventTypeRemoteThreadCreate        EventType = "remote_thread_create"
	EventTypeRemount                   EventType = "remount"
	EventTypeRename                    EventType = "rename"
	EventTypeScreensharingAttach       EventType = "screensharing_attach"
	EventTypeScreensharingDetach       EventType = "screensharing_detach"
	EventTypeSearchfs                  EventType = "searchfs"
	EventTypeSetacl                    EventType = "setacl"
	EventTypeSetattrlist               EventType = "setattrlist"
	EventTypeSetegid                   EventType = "setegid"
	EventTypeSeteuid                   EventType = "seteuid"
	EventTypeSetextattr                EventType = "setextattr"
	EventTypeSetflags                  EventType = "setflags"
	EventTypeSetgid                    EventType = "setgid"
	EventTypeSetmode                   EventType = "setmode"
	EventTypeSetowner                  EventType = "setowner"
	EventTypeSetregid                  EventType = "setregid"
	EventTypeSetreuid                  EventType = "setreuid"
	EventTypeSettime                   EventType = "settime"
	EventTypeSetuid                    EventType = "setuid"
	EventTypeSignal                    EventType = "signal"
	EventTypeStat                      EventType = "stat"
	EventTypeSu                        EventType = "su"
	EventTypeSudo                      EventType = "sudo"
	EventTypeTccModify                 EventType = "tcc_modify"
	EventTypeTrace                     EventType = "trace"
	EventTypeTruncate                  EventType = "truncate"
	EventTypeUipcBind                  EventType = "uipc_bind"
	EventTypeUipcConnect               EventType = "uipc_connect"
	EventTypeUnlink                    EventType = "unlink"
	EventTypeUnmount                   EventType = "unmount"
	EventTypeUtimes                    EventType = "utimes"
	EventTypeWrite                     EventType = "write"
	EventTypeXpMalwareDetected         EventType = "xp_malware_detected"
	EventTypeXpMalwareRemediated       EventType = "xp_malware_remediated"
	EventTypeXpcConnect                EventType = "xpc_connect"
)

// ValidEventTypes contains all valid event types as a set for validation
var ValidEventTypes = map[EventType]bool{
	EventTypeAccess:                    true,
	EventTypeAuthentication:            true,
	EventTypeAuthorizationJudgement:    true,
	EventTypeAuthorizationPetition:     true,
	EventTypeBtmLaunchItemAdd:          true,
	EventTypeBtmLaunchItemRemove:       true,
	EventTypeChdir:                     true,
	EventTypeChroot:                    true,
	EventTypeClone:                     true,
	EventTypeClose:                     true,
	EventTypeCopyfile:                  true,
	EventTypeCreate:                    true,
	EventTypeCsInvalidated:             true,
	EventTypeDeleteextattr:             true,
	EventTypeDup:                       true,
	EventTypeExchangedata:              true,
	EventTypeExec:                      true,
	EventTypeExit:                      true,
	EventTypeFcntl:                     true,
	EventTypeFileProviderMaterialize:   true,
	EventTypeFileProviderUpdate:        true,
	EventTypeFork:                      true,
	EventTypeFsgetpath:                 true,
	EventTypeGatekeeperUserOverride:    true,
	EventTypeGetTask:                   true,
	EventTypeGetTaskInspect:            true,
	EventTypeGetTaskName:               true,
	EventTypeGetTaskRead:               true,
	EventTypeGetattrlist:               true,
	EventTypeGetextattr:                true,
	EventTypeIokitOpen:                 true,
	EventTypeKextload:                  true,
	EventTypeKextunload:                true,
	EventTypeLink:                      true,
	EventTypeListextattr:               true,
	EventTypeLoginLogin:                true,
	EventTypeLoginLogout:               true,
	EventTypeLookup:                    true,
	EventTypeLwSessionLock:             true,
	EventTypeLwSessionLogin:            true,
	EventTypeLwSessionLogout:           true,
	EventTypeLwSessionUnlock:           true,
	EventTypeMmap:                      true,
	EventTypeMount:                     true,
	EventTypeMprotect:                  true,
	EventTypeOdAttributeSet:            true,
	EventTypeOdAttributeValueAdd:       true,
	EventTypeOdAttributeValueRemove:   true,
	EventTypeOdCreateGroup:             true,
	EventTypeOdCreateUser:              true,
	EventTypeOdDeleteGroup:             true,
	EventTypeOdDeleteUser:              true,
	EventTypeOdDisableUser:             true,
	EventTypeOdEnableUser:              true,
	EventTypeOdGroupAdd:                true,
	EventTypeOdGroupRemove:             true,
	EventTypeOdGroupSet:                true,
	EventTypeOdModifyPassword:          true,
	EventTypeOpen:                      true,
	EventTypeOpensshLogin:              true,
	EventTypeOpensshLogout:             true,
	EventTypeProcCheck:                 true,
	EventTypeProcSuspendResume:         true,
	EventTypeProfileAdd:                true,
	EventTypeProfileRemove:             true,
	EventTypePtyClose:                  true,
	EventTypePtyGrant:                  true,
	EventTypeReaddir:                   true,
	EventTypeReadlink:                  true,
	EventTypeRemoteThreadCreate:        true,
	EventTypeRemount:                   true,
	EventTypeRename:                    true,
	EventTypeScreensharingAttach:       true,
	EventTypeScreensharingDetach:       true,
	EventTypeSearchfs:                  true,
	EventTypeSetacl:                    true,
	EventTypeSetattrlist:               true,
	EventTypeSetegid:                   true,
	EventTypeSeteuid:                   true,
	EventTypeSetextattr:                true,
	EventTypeSetflags:                  true,
	EventTypeSetgid:                    true,
	EventTypeSetmode:                   true,
	EventTypeSetowner:                  true,
	EventTypeSetregid:                  true,
	EventTypeSetreuid:                  true,
	EventTypeSettime:                   true,
	EventTypeSetuid:                    true,
	EventTypeSignal:                    true,
	EventTypeStat:                      true,
	EventTypeSu:                        true,
	EventTypeSudo:                      true,
	EventTypeTccModify:                 true,
	EventTypeTrace:                     true,
	EventTypeTruncate:                  true,
	EventTypeUipcBind:                  true,
	EventTypeUipcConnect:               true,
	EventTypeUnlink:                    true,
	EventTypeUnmount:                   true,
	EventTypeUtimes:                    true,
	EventTypeWrite:                     true,
	EventTypeXpMalwareDetected:         true,
	EventTypeXpMalwareRemediated:       true,
	EventTypeXpcConnect:                true,
}

// String returns the string representation of the event type
func (e EventType) String() string {
	return string(e)
}

// IsValid checks if the event type is valid
func (e EventType) IsValid() bool {
	return ValidEventTypes[e]
}

// ParseEventType parses a string into an EventType, returning an error if invalid
func ParseEventType(s string) (EventType, error) {
	et := EventType(s)
	if !et.IsValid() {
		return "", fmt.Errorf("invalid event type: %s", s)
	}
	return et, nil
}

// EventTypeSet is a type-safe set of event types
type EventTypeSet map[EventType]bool

// NewEventTypeSet creates a new EventTypeSet from a slice of event types
func NewEventTypeSet(types []EventType) EventTypeSet {
	set := make(EventTypeSet)
	for _, t := range types {
		set[t] = true
	}
	return set
}

// Contains checks if the set contains an event type
func (s EventTypeSet) Contains(et EventType) bool {
	return s[et]
}

// ToSlice converts the set to a slice of strings for command arguments
func (s EventTypeSet) ToSlice() []string {
	result := make([]string, 0, len(s))
	for et := range s {
		result = append(result, et.String())
	}
	return result
}
