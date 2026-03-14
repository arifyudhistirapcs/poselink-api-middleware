package mapper

import (
	"testing"
)

func TestNewInMemoryMapper(t *testing.T) {
	config := map[string]string{
		"M001:T001": "SN12345",
		"M002:T002": "SN67890",
	}

	mapper := NewInMemoryMapper(config)

	if mapper == nil {
		t.Fatal("NewInMemoryMapper returned nil")
	}

	if mapper.mappings == nil {
		t.Fatal("mapper.mappings is nil")
	}

	if len(mapper.mappings) != 2 {
		t.Errorf("expected 2 mappings, got %d", len(mapper.mappings))
	}
}

func TestGetSerialNumber_ValidCombination(t *testing.T) {
	config := map[string]string{
		"M001:T001": "SN12345",
		"M002:T002": "SN67890",
		"M003:T003": "SN11111",
	}

	mapper := NewInMemoryMapper(config)

	tests := []struct {
		name           string
		mid            string
		tid            string
		expectedSerial string
	}{
		{"First mapping", "M001", "T001", "SN12345"},
		{"Second mapping", "M002", "T002", "SN67890"},
		{"Third mapping", "M003", "T003", "SN11111"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serial, err := mapper.GetSerialNumber(tt.mid, tt.tid)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if serial != tt.expectedSerial {
				t.Errorf("expected serial %s, got %s", tt.expectedSerial, serial)
			}
		})
	}
}

func TestGetSerialNumber_UnknownCombination(t *testing.T) {
	config := map[string]string{
		"M001:T001": "SN12345",
	}

	mapper := NewInMemoryMapper(config)

	tests := []struct {
		name string
		mid  string
		tid  string
	}{
		{"Unknown MID", "M999", "T001"},
		{"Unknown TID", "M001", "T999"},
		{"Both unknown", "M999", "T999"},
		{"Empty MID", "", "T001"},
		{"Empty TID", "M001", ""},
		{"Both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serial, err := mapper.GetSerialNumber(tt.mid, tt.tid)
			if err == nil {
				t.Errorf("expected error for unknown combination, got serial: %s", serial)
			}
			if serial != "" {
				t.Errorf("expected empty serial for error case, got: %s", serial)
			}
		})
	}
}

func TestGetSerialNumber_ErrorMessage(t *testing.T) {
	config := map[string]string{
		"M001:T001": "SN12345",
	}

	mapper := NewInMemoryMapper(config)

	_, err := mapper.GetSerialNumber("M999", "T999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMsg := "unknown mid/tid combination: M999:T999"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestGetSerialNumber_EmptyConfig(t *testing.T) {
	mapper := NewInMemoryMapper(map[string]string{})

	_, err := mapper.GetSerialNumber("M001", "T001")
	if err == nil {
		t.Error("expected error for empty config, got nil")
	}
}
