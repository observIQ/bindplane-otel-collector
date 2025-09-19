package macosunifiedloggingencodingextension

import (
	"fmt"
	"testing"
)

func TestDebugHeaderParsing(t *testing.T) {
	header, err := ParseHeaderChunk(headerTestData)
	if err != nil {
		t.Fatalf("Error parsing header: %v", err)
	}

	fmt.Printf("BootUUID: %s (expected: C320B8CE97FA4DA59F317D392E389CEA)\n", header.BootUUID)
	fmt.Printf("SubChunkTag4: %d (0x%X) (expected: 24834 / 0x6102)\n", header.SubChunkTag4, header.SubChunkTag4)
	fmt.Printf("SubChunkDataSize4: %d (expected: 48)\n", header.SubChunkDataSize4)
	fmt.Printf("TimezonePath: %s (expected: /var/db/timezone/zoneinfo/America/New_York)\n", header.TimezonePath)
}
