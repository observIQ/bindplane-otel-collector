//go:build windows

package etw

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Define our own version of the EnableTrace function that works with simple parameters
// This avoids the need to create the exact same struct as in the advapi32 package
func enableProvider(handle syscall.Handle, providerGUID windows.GUID, keywordsAll uint64) error {
	// Get the advapi32 DLL
	advapi32 := windows.NewLazySystemDLL("advapi32.dll")
	enableTraceEx2 := advapi32.NewProc("EnableTraceEx2")

	// Define the constants we need
	const (
		EVENT_CONTROL_CODE_ENABLE_PROVIDER = 1
		TRACE_LEVEL_INFORMATION            = 4
	)

	// Create a simple version parameters struct with only the Version field
	type enableTraceParameters struct {
		Version uint32
	}
	params := enableTraceParameters{Version: 2}

	// Call the DLL function directly
	r1, _, _ := enableTraceEx2.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&providerGUID)),
		uintptr(EVENT_CONTROL_CODE_ENABLE_PROVIDER),
		uintptr(TRACE_LEVEL_INFORMATION),
		uintptr(keywordsAll),             // matchAnyKeyword
		0,                                // matchAllKeyword (none)
		0,                                // timeout (none)
		uintptr(unsafe.Pointer(&params)), // Parameters struct
	)

	if r1 != 0 {
		return syscall.Errno(r1)
	}

	return nil
}
