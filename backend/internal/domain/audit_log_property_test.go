// Feature: audit-log-revamp, Property 1: AuditLog field round-trip
package domain

import (
	"encoding/json"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

// validAuditActions contains all valid AuditAction constants for generation
var validAuditActions = []interface{}{
	AuditActionCreate,
	AuditActionUpdate,
	AuditActionDelete,
	AuditActionApprove,
	AuditActionReject,
	AuditActionBulkApprove,
	AuditActionBulkReject,
	AuditActionBulkReset,
	AuditActionBulkCreate,
	AuditActionLogin,
	AuditActionLogout,
	AuditActionChangePassword,
	AuditActionView,
	AuditActionExport,
	AuditActionImport,
	AuditActionSettle,
}

// validEntityTypes contains all valid EntityType constants for generation
var validEntityTypes = []interface{}{
	EntityTypeUser,
	EntityTypeProject,
	EntityTypeEmployee,
	EntityTypeProjectEmployee,
	EntityTypeProjectUser,
	EntityTypeEmployeeUser,
	EntityTypeBank,
	EntityTypePayrate,
	EntityTypeTimesheet,
	EntityTypePayroll,
	EntityTypeLedgerEntry,
	EntityTypeSettings,
	EntityTypeAsset,
	EntityTypeTransaction,
	EntityTypeLoan,
	EntityTypeLender,
}

// genAuditAction generates a random valid AuditAction
func genAuditAction() gopter.Gen {
	return gen.OneConstOf(validAuditActions...)
}

// genEntityType generates a random valid EntityType
func genEntityType() gopter.Gen {
	return gen.OneConstOf(validEntityTypes...)
}

// genNullableUint generates a *uint that is either nil or a positive uint
func genNullableUint() gopter.Gen {
	return gen.OneGenOf(
		gen.Const((*uint)(nil)),
		gen.UInt().Map(func(v uint) *uint {
			if v == 0 {
				v = 1
			}
			return &v
		}),
	)
}

// genNullableString generates a *string that is either nil or a non-empty string
func genNullableString() gopter.Gen {
	return gen.OneGenOf(
		gen.Const((*string)(nil)),
		gen.AlphaString().SuchThat(func(s string) bool {
			return len(s) > 0
		}).Map(func(s string) *string {
			return &s
		}),
	)
}

// genMetadataString generates a *string containing valid JSON or nil
func genMetadataString() gopter.Gen {
	return gen.OneGenOf(
		gen.Const((*string)(nil)),
		gen.AlphaString().Map(func(key string) *string {
			if key == "" {
				key = "key"
			}
			m := map[string]interface{}{key: "value"}
			b, _ := json.Marshal(m)
			s := string(b)
			return &s
		}),
	)
}

// genAuditLog generates a random AuditLog with all structured fields populated
func genAuditLog() gopter.Gen {
	return gopter.CombineGens(
		genAuditAction(),
		genEntityType(),
		genNullableUint(),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		genNullableString(),
		genNullableString(),
		genNullableString(),
		genMetadataString(),
		gen.UInt().Map(func(v uint) uint {
			if v == 0 {
				return 1
			}
			return v
		}),
	).Map(func(values []interface{}) *AuditLog {
		action := values[0].(AuditAction)
		entityType := values[1].(EntityType)
		entityID := values[2].(*uint)
		message := values[3].(string)
		ipAddress := values[4].(*string)
		browser := values[5].(*string)
		platform := values[6].(*string)
		metadata := values[7].(*string)
		userID := values[8].(uint)

		return &AuditLog{
			Action:     action,
			EntityType: entityType,
			EntityID:   entityID,
			Message:    message,
			IPAddress:  ipAddress,
			Browser:    browser,
			Platform:   platform,
			Metadata:   metadata,
			UserID:     userID,
		}
	})
}

// nullableStringEqual compares two *string values for equality
func nullableStringEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// nullableUintEqual compares two *uint values for equality
func nullableUintEqual(a, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// TestProperty1_AuditLogFieldRoundTrip validates that AuditLog fields survive
// a JSON marshal/unmarshal cycle with all structured fields intact.
//
// Validates: Requirements 1.1
func TestProperty1_AuditLogFieldRoundTrip(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParametersWithSeed(1234))
	properties.Property("AuditLog fields survive JSON round-trip", prop.ForAll(
		func(original *AuditLog) bool {
			// Marshal to JSON
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}

			// Unmarshal back
			var restored AuditLog
			if err := json.Unmarshal(data, &restored); err != nil {
				return false
			}

			// Assert all structured fields are equal
			return original.Action == restored.Action &&
				original.EntityType == restored.EntityType &&
				nullableUintEqual(original.EntityID, restored.EntityID) &&
				original.Message == restored.Message &&
				nullableStringEqual(original.IPAddress, restored.IPAddress) &&
				nullableStringEqual(original.Browser, restored.Browser) &&
				nullableStringEqual(original.Platform, restored.Platform) &&
				nullableStringEqual(original.Metadata, restored.Metadata) &&
				original.UserID == restored.UserID
		},
		genAuditLog(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty1_AuditLogFieldRoundTrip_Assertions uses testify assertions
// for clearer failure messages on specific field mismatches.
//
// Validates: Requirements 1.1
func TestProperty1_AuditLogFieldRoundTrip_Assertions(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(5678)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("AuditLog Action field survives JSON round-trip", prop.ForAll(
		func(original *AuditLog) bool {
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}
			var restored AuditLog
			if err := json.Unmarshal(data, &restored); err != nil {
				return false
			}
			return original.Action == restored.Action
		},
		genAuditLog(),
	))

	properties.Property("AuditLog EntityType field survives JSON round-trip", prop.ForAll(
		func(original *AuditLog) bool {
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}
			var restored AuditLog
			if err := json.Unmarshal(data, &restored); err != nil {
				return false
			}
			return original.EntityType == restored.EntityType
		},
		genAuditLog(),
	))

	properties.Property("AuditLog UserID field survives JSON round-trip", prop.ForAll(
		func(original *AuditLog) bool {
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}
			var restored AuditLog
			if err := json.Unmarshal(data, &restored); err != nil {
				return false
			}
			return original.UserID == restored.UserID
		},
		genAuditLog(),
	))

	properties.Property("AuditLog Message field survives JSON round-trip", prop.ForAll(
		func(original *AuditLog) bool {
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}
			var restored AuditLog
			if err := json.Unmarshal(data, &restored); err != nil {
				return false
			}
			return original.Message == restored.Message
		},
		genAuditLog(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty1_AuditLogAllFieldsRoundTrip_MinIterations ensures minimum 100 iterations
// with explicit parameter configuration.
//
// Validates: Requirements 1.1
func TestProperty1_AuditLogAllFieldsRoundTrip_MinIterations(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(9999)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("all AuditLog structured fields survive JSON round-trip", prop.ForAll(
		func(original *AuditLog) bool {
			data, err := json.Marshal(original)
			if err != nil {
				t.Logf("marshal error: %v", err)
				return false
			}

			var restored AuditLog
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Logf("unmarshal error: %v", err)
				return false
			}

			assert.Equal(t, original.Action, restored.Action, "Action mismatch")
			assert.Equal(t, original.EntityType, restored.EntityType, "EntityType mismatch")
			assert.Equal(t, original.UserID, restored.UserID, "UserID mismatch")
			assert.Equal(t, original.Message, restored.Message, "Message mismatch")

			if original.EntityID == nil {
				assert.Nil(t, restored.EntityID, "EntityID should be nil")
			} else {
				assert.NotNil(t, restored.EntityID, "EntityID should not be nil")
				assert.Equal(t, *original.EntityID, *restored.EntityID, "EntityID value mismatch")
			}

			if original.IPAddress == nil {
				assert.Nil(t, restored.IPAddress, "IPAddress should be nil")
			} else {
				assert.NotNil(t, restored.IPAddress, "IPAddress should not be nil")
				assert.Equal(t, *original.IPAddress, *restored.IPAddress, "IPAddress value mismatch")
			}

			if original.Browser == nil {
				assert.Nil(t, restored.Browser, "Browser should be nil")
			} else {
				assert.NotNil(t, restored.Browser, "Browser should not be nil")
				assert.Equal(t, *original.Browser, *restored.Browser, "Browser value mismatch")
			}

			if original.Platform == nil {
				assert.Nil(t, restored.Platform, "Platform should be nil")
			} else {
				assert.NotNil(t, restored.Platform, "Platform should not be nil")
				assert.Equal(t, *original.Platform, *restored.Platform, "Platform value mismatch")
			}

			if original.Metadata == nil {
				assert.Nil(t, restored.Metadata, "Metadata should be nil")
			} else {
				assert.NotNil(t, restored.Metadata, "Metadata should not be nil")
				assert.Equal(t, *original.Metadata, *restored.Metadata, "Metadata value mismatch")
			}

			return !t.Failed()
		},
		genAuditLog(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: audit-log-revamp, Property 2: Empty action is rejected
// TestProperty2_EmptyActionRejected verifies that BeforeCreate returns a validation error
// whenever the AuditLog Action field is empty.
//
// Validates: Requirements 1.2
func TestProperty2_EmptyActionRejected(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(2222)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("BeforeCreate rejects AuditLog with empty Action", prop.ForAll(
		func(log *AuditLog) bool {
			log.Action = ""
			err := log.BeforeCreate(nil)
			return err != nil
		},
		genAuditLog(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: audit-log-revamp, Property 3: Empty entity_type is rejected
// TestProperty3_EmptyEntityTypeRejected verifies that BeforeCreate returns a validation error
// whenever the AuditLog EntityType field is empty.
//
// Validates: Requirements 1.3
func TestProperty3_EmptyEntityTypeRejected(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(3333)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("BeforeCreate rejects AuditLog with empty EntityType", prop.ForAll(
		func(log *AuditLog) bool {
			log.EntityType = ""
			err := log.BeforeCreate(nil)
			return err != nil
		},
		genAuditLog(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: audit-log-revamp, Property 14: Metadata serialization round-trip
// TestProperty14_MetadataSerializationRoundTrip verifies that SetMetadata followed by
// GetMetadata returns an equivalent map, accounting for JSON's int-to-float64 conversion.
//
// Validates: Requirements 9.5
func TestProperty14_MetadataSerializationRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(14141)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// genMetadataMap generates a random map[string]interface{} with string keys
	// and string or bool values (ints are avoided because JSON round-trip converts
	// them to float64, making JSON-level comparison the correct approach).
	genMetadataMap := gopter.CombineGens(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString(),
		gen.AlphaString(),
		gen.Bool(),
	).Map(func(values []interface{}) map[string]interface{} {
		k1 := values[0].(string)
		k2 := values[1].(string)
		k3 := values[2].(string)
		v1 := values[3].(string)
		v2 := values[4].(string)
		v3 := values[5].(bool)
		return map[string]interface{}{
			k1: v1,
			k2: v2,
			k3: v3,
		}
	})

	properties.Property("SetMetadata then GetMetadata returns equivalent map", prop.ForAll(
		func(m map[string]interface{}) bool {
			log := &AuditLog{
				Action:     AuditActionCreate,
				EntityType: EntityTypeUser,
				UserID:     1,
				Message:    "test",
			}

			if err := log.SetMetadata(m); err != nil {
				return false
			}

			got, err := log.GetMetadata()
			if err != nil {
				return false
			}

			// JSON round-trip converts int values to float64; compare via JSON representation
			originalJSON, err := json.Marshal(m)
			if err != nil {
				return false
			}
			gotJSON, err := json.Marshal(got)
			if err != nil {
				return false
			}

			return string(originalJSON) == string(gotJSON)
		},
		genMetadataMap,
	))

	properties.Property("SetMetadata(nil) results in GetMetadata returning nil, nil", prop.ForAll(
		func(log *AuditLog) bool {
			if err := log.SetMetadata(nil); err != nil {
				return false
			}
			got, err := log.GetMetadata()
			return got == nil && err == nil
		},
		genAuditLog(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
