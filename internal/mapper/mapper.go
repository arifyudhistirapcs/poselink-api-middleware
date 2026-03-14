package mapper

import "fmt"

// MIDTIDMapper provides mapping from MID/TID combinations to EDC device serial numbers
type MIDTIDMapper interface {
	// GetSerialNumber returns the serial number for a given MID and TID combination
	// Returns an error if the combination is not found in the configured mappings
	GetSerialNumber(mid, tid string) (string, error)
}

// InMemoryMapper implements MIDTIDMapper using an in-memory map
type InMemoryMapper struct {
	mappings map[string]string // key format: "mid:tid", value: serial_number
}

// NewInMemoryMapper creates a new InMemoryMapper with the provided configuration map
// The config map should use "mid:tid" as keys and serial numbers as values
func NewInMemoryMapper(config map[string]string) *InMemoryMapper {
	return &InMemoryMapper{
		mappings: config,
	}
}

// GetSerialNumber looks up the serial number for the given MID and TID combination
// Returns a descriptive error if the combination is not found
func (m *InMemoryMapper) GetSerialNumber(mid, tid string) (string, error) {
	key := fmt.Sprintf("%s:%s", mid, tid)
	serialNumber, exists := m.mappings[key]
	if !exists {
		return "", fmt.Errorf("unknown mid/tid combination: %s", key)
	}
	return serialNumber, nil
}
