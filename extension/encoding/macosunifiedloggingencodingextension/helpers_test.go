package macosunifiedloggingencodingextension

import (
	"testing"
)

func TestExtractStringSize(t *testing.T) {
	data := []byte{55, 57, 54, 46, 49, 48, 48, 0}
	size := uint64(8)
	_, result, err := ExtractStringSize(data, size)
	if err != nil {
		t.Fatalf("failed to extract string size: %v", err)
	}
	if result != "796.100" {
		t.Fatalf("expected '796.100', got '%s'", result)
	}
}
