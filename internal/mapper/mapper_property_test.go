package mapper

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: payment-middleware, Property 3: MID/TID Mapping Success
// Validates: Requirements 1.4, 7.2
func TestProperty_MIDTIDMappingSuccess(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("valid MID/TID combinations should return correct serial number", prop.ForAll(
		func(mid, tid, serial string) bool {
			// Skip empty values
			if strings.TrimSpace(mid) == "" || strings.TrimSpace(tid) == "" || strings.TrimSpace(serial) == "" {
				return true
			}

			// Create mapper with the mapping
			mappings := map[string]string{
				fmt.Sprintf("%s:%s", mid, tid): serial,
			}
			mapper := NewInMemoryMapper(mappings)

			// Lookup should succeed
			result, err := mapper.GetSerialNumber(mid, tid)
			if err != nil {
				return false
			}

			// Result should match the configured serial number
			return result == serial
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// Feature: payment-middleware, Property 4: Unknown MID/TID Rejection
// Validates: Requirements 1.5, 7.3, 8.3
func TestProperty_UnknownMIDTIDRejection(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("unknown MID/TID combinations should return error", prop.ForAll(
		func(mid, tid string) bool {
			// Skip empty values
			if strings.TrimSpace(mid) == "" || strings.TrimSpace(tid) == "" {
				return true
			}

			// Create mapper with no mappings
			mapper := NewInMemoryMapper(map[string]string{})

			// Lookup should fail
			_, err := mapper.GetSerialNumber(mid, tid)
			return err != nil
		},
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// Feature: payment-middleware, Property 3 Extended: Multiple Mappings
// Validates: Requirements 7.2, 7.4
func TestProperty_MultipleMappings(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 20
	properties := gopter.NewProperties(params)
	properties.Property("mapper should handle multiple mappings correctly", prop.ForAll(
		func(mappings map[string]string) bool {
			if len(mappings) == 0 {
				return true
			}

			mapper := NewInMemoryMapper(mappings)

			// Verify all mappings work
			for key, expectedSerial := range mappings {
				parts := strings.Split(key, ":")
				if len(parts) != 2 {
					continue
				}

				mid, tid := parts[0], parts[1]
				if strings.TrimSpace(mid) == "" || strings.TrimSpace(tid) == "" {
					continue
				}

				result, err := mapper.GetSerialNumber(mid, tid)
				if err != nil {
					return false
				}

				if result != expectedSerial {
					return false
				}
			}

			return true
		},
		gen.MapOf(
			gen.Identifier().Map(func(id string) string {
				return fmt.Sprintf("%s:%s", id, id+"_tid")
			}),
			gen.Identifier(),
		),
	))

	properties.TestingRun(t)
}
