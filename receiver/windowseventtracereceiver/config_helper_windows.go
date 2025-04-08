//go:build windows

package windowseventtracereceiver

import "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"

func (l TraceLevelString) toTraceLevel() advapi32.TraceLevel {
	switch l {
	case LevelVerbose:
		return advapi32.TRACE_LEVEL_VERBOSE
	case LevelInformational:
		return advapi32.TRACE_LEVEL_INFORMATION
	case LevelWarning:
		return advapi32.TRACE_LEVEL_WARNING
	case LevelError:
		return advapi32.TRACE_LEVEL_ERROR
	case LevelCritical:
		return advapi32.TRACE_LEVEL_CRITICAL
	case LevelNone:
		return advapi32.TRACE_LEVEL_NONE
	default:
		return advapi32.TRACE_LEVEL_INFORMATION
	}
}
