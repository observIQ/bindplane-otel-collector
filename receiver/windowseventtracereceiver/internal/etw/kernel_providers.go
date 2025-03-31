//go:build windows
// +build windows

package etw

import "strings"

type ProviderDefinition struct {
	Name   string
	Kernel bool
	GUID   string
	Flags  uint32
}

var (
	KernelProviders = []ProviderDefinition{
		// https://docs.microsoft.com/en-us/windows/win32/etw/alpc
		{Name: "ALPC",
			Kernel: true,
			GUID:   "{45d8cccd-539f-4b72-a8b7-5c683142609a}",
			Flags:  EVENT_TRACE_FLAG_ALPC},
		//{Name: "ApplicationVerifier", Kernel: true, GUID: "{78d14f17-0105-46d7-bfff-6fbea2f3f358}"},
		{Name: "DbgPrint",
			Kernel: true,
			GUID:   "{13976d09-a327-438c-950b-7f03192815c7}",
			Flags:  EVENT_TRACE_FLAG_DBGPRINT},
		// https://docs.microsoft.com/en-us/windows/win32/etw/diskio
		{Name: "DiskIo",
			Kernel: true,
			GUID:   "{3d6fa8d4-fe05-11d0-9dda-00c04fd7ba7c}",
			Flags:  EVENT_TRACE_FLAG_DISK_IO},
		{Name: "DiskIoInit",
			Kernel: true,
			GUID:   "{3d6fa8d4-fe05-11d0-9dda-00c04fd7ba7c}",
			Flags:  EVENT_TRACE_FLAG_DISK_IO_INIT},
		{Name: "Driver",
			Kernel: true,
			GUID:   "{3d6fa8d4-fe05-11d0-9dda-00c04fd7ba7c}",
			Flags:  EVENT_TRACE_FLAG_DRIVER},
		//{Name: "DiskPerf", Kernel: true, GUID: "{bdd865d1-d7c1-11d0-a501-00a0c9062910}"},
		//{Name: "DriverVerifier", Kernel: true, GUID: "{d56ca431-61bf-4904-a621-00e0381e4dde"},
		//{Name: "EventLog", Kernel: true, GUID: "{b16f9f5e-b3da-4027-9318-adf2b79df73b}"},
		//{Name: "EventTraceConfig", Kernel: true, GUID: "{01853a65-418f-4f36-aefc-dc0f1d2fd235}"},
		// https://docs.microsoft.com/en-us/windows/win32/etw/fileio
		{Name: "FileIo",
			Kernel: true,
			GUID:   "{90cbdc39-4a3e-11d1-84f4-0000f80464e3}",
			Flags:  EVENT_TRACE_FLAG_FILE_IO},
		{Name: "FileIoInit",
			Kernel: true,
			GUID:   "{90cbdc39-4a3e-11d1-84f4-0000f80464e3}",
			Flags:  EVENT_TRACE_FLAG_FILE_IO_INIT},
		//{Name: "GenericMessage", Kernel: true, GUID: "{8d40301f-ab4a-11d2-9a93-00805f85d7c6}"},
		//{Name: "GlobalLogger", Kernel: true, GUID: "{e8908abc-aa84-11d2-9a93-00805f85d7c6}"},
		//{Name: "HardFault", Kernel: true, GUID: "{3d6fa8d2-fe05-11d0-9dda-00c04fd7ba7c}"},
		// https://docs.microsoft.com/en-us/windows/win32/etw/image
		{Name: "ImageLoad",
			Kernel: true,
			GUID:   "{2cb15d1d-5fc1-11d2-abe1-00a0c911f518}",
			Flags:  EVENT_TRACE_FLAG_IMAGE_LOAD},
		//{Name: "MsSystemInformation", Kernel: true, GUID: "{98a2b9d7-94dd-496a-847e-67a5557a59f2}"},
		// https://docs.microsoft.com/en-us/windows/win32/etw/pagefault-v2
		{Name: "MemoryPageFault",
			Kernel: true,
			GUID:   "{3d6fa8d3-fe05-11d0-9dda-00c04fd7ba7c}",
			Flags:  EVENT_TRACE_FLAG_MEMORY_PAGE_FAULTS},
		{Name: "MemoryHardFault",
			Kernel: true,
			GUID:   "{3d6fa8d3-fe05-11d0-9dda-00c04fd7ba7c}",
			Flags:  EVENT_TRACE_FLAG_MEMORY_HARD_FAULTS},
		{Name: "VirtualAlloc",
			Kernel: true,
			GUID:   "{3d6fa8d3-fe05-11d0-9dda-00c04fd7ba7c}",
			Flags:  EVENT_TRACE_FLAG_VIRTUAL_ALLOC},
		// https://docs.microsoft.com/en-us/windows/win32/etw/process
		{Name: "DPC",
			Kernel: true,
			GUID:   "{ce1dbfb4-137e-4da6-87b0-3f59aa102cbc}",
			Flags:  EVENT_TRACE_FLAG_DPC},
		{Name: "Interrupt",
			Kernel: true,
			GUID:   "{ce1dbfb4-137e-4da6-87b0-3f59aa102cbc}",
			Flags:  EVENT_TRACE_FLAG_INTERRUPT},
		{Name: "Profile",
			Kernel: true,
			GUID:   "{ce1dbfb4-137e-4da6-87b0-3f59aa102cbc}",
			Flags:  EVENT_TRACE_FLAG_PROFILE},
		{Name: "Syscall",
			Kernel: true,
			GUID:   "{ce1dbfb4-137e-4da6-87b0-3f59aa102cbc}",
			Flags:  EVENT_TRACE_FLAG_SYSTEMCALL},
		// https://docs.microsoft.com/en-us/windows/win32/etw/process
		{Name: "Process",
			Kernel: true,
			GUID:   "{3d6fa8d0-fe05-11d0-9dda-00c04fd7ba7c}",
			Flags:  EVENT_TRACE_FLAG_PROCESS},
		{Name: "ProcessCounters",
			Kernel: true,
			GUID:   "{3d6fa8d0-fe05-11d0-9dda-00c04fd7ba7c}",
			Flags:  EVENT_TRACE_FLAG_PROCESS_COUNTERS},
		// https://docs.microsoft.com/en-us/windows/win32/etw/registry
		{Name: "Registry",
			Kernel: true,
			GUID:   "{ae53722e-c863-11d2-8659-00c04fa321a1}",
			Flags:  EVENT_TRACE_FLAG_REGISTRY},
		// https://docs.microsoft.com/en-us/windows/win32/etw/splitio
		{Name: "SplitIo",
			Kernel: true,
			GUID:   "{d837ca92-12b9-44a5-ad6a-3a65b3578aa8}",
			Flags:  EVENT_TRACE_FLAG_SPLIT_IO},
		// https://docs.microsoft.com/en-us/windows/win32/etw/tcpip
		{Name: "TcpIp",
			Kernel: true,
			GUID:   "{9a280ac0-c8e0-11d1-84e2-00c04fb998a2}",
			Flags:  EVENT_TRACE_FLAG_NETWORK_TCPIP},
		//{Name: "ThermalZone", Kernel: true, GUID: "{a1bc18c0-a7c8-11d1-bf3c-00a0c9062910}"},
		// https://docs.microsoft.com/en-us/windows/win32/etw/thread
		{Name: "Thread",
			Kernel: true,
			GUID:   "{3d6fa8d1-fe05-11d0-9dda-00c04fd7ba7c}",
			Flags:  EVENT_TRACE_FLAG_THREAD},
		//{Name: "TraceError", Kernel: true, GUID: "{398191dc-2da7-11d3-8b98-00805f85d7c6}"},
		// https://docs.microsoft.com/en-us/windows/win32/etw/udpip
		{Name: "UdpIp",
			Kernel: true, GUID: "{bf3a50c5-a9c9-4988-a005-2df0b7c80f80}",
			Flags: EVENT_TRACE_FLAG_NETWORK_TCPIP},

		//{Name: "WmiEventLogger", Kernel: true, GUID: "{44608a51-1851-4456-98b2-b300e931ee41}"}
	}
)

func IsKernelProvider(term string) bool {
	for _, pd := range KernelProviders {
		if strings.EqualFold(term, pd.Name) || term == pd.GUID {
			return true
		}
	}
	return false
}

func GetKernelProviderFlags(terms ...string) (flags uint32) {
	for _, t := range terms {
		for _, pd := range KernelProviders {
			if strings.EqualFold(t, pd.Name) || t == pd.GUID {
				flags |= pd.Flags
			}
		}

	}
	return
}
