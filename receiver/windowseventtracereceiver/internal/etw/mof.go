//go:build windows
// +build windows

package etw

type MofClass struct {
	// Class name
	Name string
	// Serves as base to compute event id
	BaseId uint16
}

var (
	// The final event id of Mof Events is computed
	// by BaseId + Opcode. As Opcode is uint8 we jump
	// BaseIds every 0xff so that we do not overlap event
	// ids between classes
	MofClassMapping = map[uint32]MofClass{
		/*45d8cccd-539f-4b72-a8b7-5c683142609a*/
		1171836109: {Name: "ALPC", BaseId: 0},
		/*78d14f17-0105-46d7-bfff-6fbea2f3f358*/
		2026983191: {Name: "ApplicationVerifier", BaseId: 255},
		/*13976d09-a327-438c-950b-7f03192815c7*/
		328690953: {Name: "DbgPrint", BaseId: 510},
		/*3d6fa8d4-fe05-11d0-9dda-00c04fd7ba7c*/
		1030727892: {Name: "DiskIo", BaseId: 765},
		/*bdd865d1-d7c1-11d0-a501-00a0c9062910*/
		3185075665: {Name: "DiskPerf", BaseId: 1020},
		/*d56ca431-61bf-4904-a621-00e0381e4dde*/
		3580666929: {Name: "DriverVerifier", BaseId: 1275},
		/*b16f9f5e-b3da-4027-9318-adf2b79df73b*/
		2976882526: {Name: "EventLog", BaseId: 1530},
		/*01853a65-418f-4f36-aefc-dc0f1d2fd235*/
		25508453: {Name: "EventTraceConfig", BaseId: 1785},
		/*90cbdc39-4a3e-11d1-84f4-0000f80464e3*/
		2429279289: {Name: "FileIo", BaseId: 2040},
		/*8d40301f-ab4a-11d2-9a93-00805f85d7c6*/
		2369794079: {Name: "GenericMessage", BaseId: 2295},
		/*e8908abc-aa84-11d2-9a93-00805f85d7c6*/
		3901786812: {Name: "GlobalLogger", BaseId: 2550},
		/*3d6fa8d2-fe05-11d0-9dda-00c04fd7ba7c*/
		1030727890: {Name: "HardFault", BaseId: 2805},
		/*2cb15d1d-5fc1-11d2-abe1-00a0c911f518*/
		749821213: {Name: "ImageLoad", BaseId: 3060},
		/*98a2b9d7-94dd-496a-847e-67a5557a59f2*/
		2560801239: {Name: "MsSystemInformation", BaseId: 3315},
		/*3d6fa8d3-fe05-11d0-9dda-00c04fd7ba7c*/
		1030727891: {Name: "PageFault", BaseId: 3570},
		/*ce1dbfb4-137e-4da6-87b0-3f59aa102cbc*/
		3458056116: {Name: "PerfInfo", BaseId: 3825},
		/*3d6fa8d0-fe05-11d0-9dda-00c04fd7ba7c*/
		1030727888: {Name: "Process", BaseId: 4080},
		/*ae53722e-c863-11d2-8659-0c04fa321a1*/
		2924704302: {Name: "Registry", BaseId: 4335},
		/*d837ca92-12b9-44a5-ad6a-3a65b3578aa8*/
		3627534994: {Name: "SplitIo", BaseId: 4590},
		/*9a280ac0-c8e0-11d1-84e2-00c04fb998a2*/
		2586315456: {Name: "TcpIp", BaseId: 4845},
		/*a1bc18c0-a7c8-11d1-bf3c-00a0c9062910*/
		2713458880: {Name: "ThermalZone", BaseId: 5100},
		/*3d6fa8d1-fe05-11d0-9dda-00c04fd7ba7c*/
		1030727889: {Name: "Thread", BaseId: 5355},
		/*398191dc-2da7-11d3-8b98-00805f85d7c6*/
		964792796: {Name: "TraceError", BaseId: 5610},
		/*bf3a50c5-a9c9-4988-a005-2df0b7c80f80*/
		3208270021: {Name: "UdpIp", BaseId: 5865},
		/*44608a51-1851-4456-98b2-b300e931ee41*/
		1147177553: {Name: "WmiEventLogger", BaseId: 6120},
		/*68fdd900-4a3e-11d1-84f4-0000f80464e3*/
		0x68fdd900: {Name: "EventTraceEvent", BaseId: 6375},
	}
)
