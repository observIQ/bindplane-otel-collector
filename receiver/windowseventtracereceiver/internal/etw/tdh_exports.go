//go:build windows
// +build windows

package etw

import (
	"syscall"
)

var (
	tdh                                            = syscall.NewLazyDLL("tdh.dll")
	tdhEnumerateProviderFieldInformation           = tdh.NewProc("TdhEnumerateProviderFieldInformation")
	tdhEnumerateProviderFilters                    = tdh.NewProc("TdhEnumerateProviderFilters")
	tdhEnumerateProviders                          = tdh.NewProc("TdhEnumerateProviders")
	tdhEnumerateRemoteWBEMProviderFieldInformation = tdh.NewProc("TdhEnumerateRemoteWBEMProviderFieldInformation")
	tdhEnumerateRemoteWBEMProviders                = tdh.NewProc("TdhEnumerateRemoteWBEMProviders")
	tdhFormatProperty                              = tdh.NewProc("TdhFormatProperty")
	tdhGetAllEventsInformation                     = tdh.NewProc("TdhGetAllEventsInformation")
	tdhGetEventInformation                         = tdh.NewProc("TdhGetEventInformation")
	tdhGetEventMapInformation                      = tdh.NewProc("TdhGetEventMapInformation")
	tdhGetProperty                                 = tdh.NewProc("TdhGetProperty")
	tdhGetPropertyOffsetAndSize                    = tdh.NewProc("TdhGetPropertyOffsetAndSize")
	tdhGetPropertySize                             = tdh.NewProc("TdhGetPropertySize")
	tdhLoadManifest                                = tdh.NewProc("TdhLoadManifest")
	tdhQueryProviderFieldInformation               = tdh.NewProc("TdhQueryProviderFieldInformation")
	tdhQueryRemoteWBEMProviderFieldInformation     = tdh.NewProc("TdhQueryRemoteWBEMProviderFieldInformation")
	tdhUnloadManifest                              = tdh.NewProc("TdhUnloadManifest")
)
