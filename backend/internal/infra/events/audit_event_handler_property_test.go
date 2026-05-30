// Feature: audit-log-revamp, Property 5: AuditEventHandler persists structured fields from event
// Feature: audit-log-revamp, Property 7: Failed login metadata is stored
// Feature: audit-log-revamp, Property 13: BulkTransfer metadata completeness
package events

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
	asynqinfra "api-server/internal/infra/asynq"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// mockAuditEnqueuer captures payloads enqueued via EnqueueAuditLogWrite.
type mockAuditEnqueuer struct {
	payloads []asynqinfra.AuditLogWritePayload
}

func (m *mockAuditEnqueuer) EnqueueAuditLogWrite(p asynqinfra.AuditLogWritePayload) error {
	m.payloads = append(m.payloads, p)
	return nil
}

// mockDomainEvent is a minimal DomainEvent implementation for Property 5.
type mockDomainEvent struct {
	action     domain.AuditAction
	entityType domain.EntityType
	entityID   uint
	userID     uint
	ipAddress  string
	userAgent  string
	message    string
}

func (e mockDomainEvent) EventType() string                { return "MockEvent" }
func (e mockDomainEvent) OccurredAt() time.Time            { return time.Now() }
func (e mockDomainEvent) AggregateID() uint                { return e.entityID }
func (e mockDomainEvent) UserID() uint                     { return e.userID }
func (e mockDomainEvent) GetAuditMessage() string          { return e.message }
func (e mockDomainEvent) GetAction() domain.AuditAction    { return e.action }
func (e mockDomainEvent) GetEntityType() domain.EntityType { return e.entityType }
func (e mockDomainEvent) GetIPAddress() string             { return e.ipAddress }
func (e mockDomainEvent) GetUserAgent() string             { return e.userAgent }

// validAuditActionsForHandler lists all valid AuditAction constants.
var validAuditActionsForHandler = []interface{}{
	domain.AuditActionCreate,
	domain.AuditActionUpdate,
	domain.AuditActionDelete,
	domain.AuditActionApprove,
	domain.AuditActionReject,
	domain.AuditActionBulkApprove,
	domain.AuditActionBulkReject,
	domain.AuditActionBulkReset,
	domain.AuditActionBulkCreate,
	domain.AuditActionLogin,
	domain.AuditActionLogout,
	domain.AuditActionChangePassword,
	domain.AuditActionView,
	domain.AuditActionExport,
	domain.AuditActionImport,
	domain.AuditActionSettle,
}

// validEntityTypesForHandler lists all valid EntityType constants.
var validEntityTypesForHandler = []interface{}{
	domain.EntityTypeUser,
	domain.EntityTypeProject,
	domain.EntityTypeEmployee,
	domain.EntityTypeProjectEmployee,
	domain.EntityTypeProjectUser,
	domain.EntityTypeEmployeeUser,
	domain.EntityTypeBank,
	domain.EntityTypePayrate,
	domain.EntityTypeTimesheet,
	domain.EntityTypePayroll,
	domain.EntityTypeLedgerEntry,
	domain.EntityTypeSettings,
	domain.EntityTypeAsset,
	domain.EntityTypeTransaction,
	domain.EntityTypeLoan,
	domain.EntityTypeLender,
}

func genHandlerAuditAction() gopter.Gen {
	return gen.OneConstOf(validAuditActionsForHandler...)
}

func genHandlerEntityType() gopter.Gen {
	return gen.OneConstOf(validEntityTypesForHandler...)
}

// genNonEmptyAlphaString generates a non-empty alpha string.
func genNonEmptyAlphaString() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 })
}

// genPositiveUint generates a uint > 0.
func genPositiveUint() gopter.Gen {
	return gen.UInt().Map(func(v uint) uint {
		if v == 0 {
			return 1
		}
		return v
	})
}

// TestProperty5_HandlerPersistsStructuredFields verifies that AuditEventHandler
// enqueues a payload whose fields exactly match the values from the domain event.
// UA parsing (Browser/Platform) is deferred to the worker and not tested here.
//
// Validates: Requirements 3.1–3.12, 4.3, 8.4
func TestProperty5_HandlerPersistsStructuredFields(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(5555)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	rawUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/147.0.0.0 Safari/537.36"

	genEvent := gopter.CombineGens(
		genHandlerAuditAction(),
		genHandlerEntityType(),
		genPositiveUint(),        // entityID
		genPositiveUint(),        // userID
		genNonEmptyAlphaString(), // ipAddress
		genNonEmptyAlphaString(), // message
	).Map(func(values []interface{}) mockDomainEvent {
		return mockDomainEvent{
			action:     values[0].(domain.AuditAction),
			entityType: values[1].(domain.EntityType),
			entityID:   values[2].(uint),
			userID:     values[3].(uint),
			ipAddress:  values[4].(string),
			userAgent:  rawUA,
			message:    values[5].(string),
		}
	})

	properties.Property("enqueued payload fields match event values", prop.ForAll(
		func(event mockDomainEvent) bool {
			enqueuer := &mockAuditEnqueuer{}
			handler := NewAuditEventHandler(enqueuer)

			if err := handler.Handle(context.Background(), event); err != nil {
				return false
			}

			if len(enqueuer.payloads) != 1 {
				return false
			}

			p := enqueuer.payloads[0]
			if p.Action != string(event.action) {
				return false
			}
			if p.EntityType != string(event.entityType) {
				return false
			}
			if p.UserID != event.userID {
				return false
			}
			if p.Message != event.message {
				return false
			}
			if p.IPAddress != event.ipAddress {
				return false
			}
			if !strings.Contains(p.UserAgent, "Chrome") {
				return false
			}
			if p.EntityID == nil || *p.EntityID != event.entityID {
				return false
			}
			return true
		},
		genEvent,
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty7_FailedLoginMetadataIsStored verifies that when a UserLoginEvent
// with Success=false is handled, the enqueued payload metadata contains
// {"success": false, "reason": "<reason>"}.
//
// Validates: Requirements 4.4, 9.2
func TestProperty7_FailedLoginMetadataIsStored(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(7777)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("failed login metadata contains success=false and reason", prop.ForAll(
		func(reason string) bool {
			enqueuer := &mockAuditEnqueuer{}
			handler := NewAuditEventHandler(enqueuer)

			event := domain.UserLoginEvent{
				BaseEvent: domain.BaseEvent{
					EventName:    "UserLogin",
					ActorUserID:  1,
					AuditMessage: "đã đăng nhập thất bại",
					Action:       domain.AuditActionLogin,
					EntityType:   domain.EntityTypeUser,
				},
				Success: false,
				Reason:  reason,
			}

			if err := handler.Handle(context.Background(), event); err != nil {
				return false
			}

			if len(enqueuer.payloads) != 1 {
				return false
			}

			p := enqueuer.payloads[0]
			if p.MetadataJSON == "" {
				return false
			}

			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(p.MetadataJSON), &meta); err != nil {
				return false
			}

			successVal, ok := meta["success"]
			if !ok {
				return false
			}
			if successVal != false {
				return false
			}

			reasonVal, ok := meta["reason"]
			if !ok {
				return false
			}
			if reasonVal != reason {
				return false
			}

			return true
		},
		genNonEmptyAlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestHandler_DropsEmitsWithUserIDZero_ExceptLogin verifies the filter rule:
// audit rows with user_id=0 are silently dropped except for login attempts
// (failed logins legitimately carry user_id=0). Cron jobs / system writes
// without an actor must not pollute the audit log.
func TestHandler_DropsEmitsWithUserIDZero_ExceptLogin(t *testing.T) {
	// Non-login event with user_id=0 → dropped.
	t.Run("drops non-login event when user_id=0", func(t *testing.T) {
		enqueuer := &mockAuditEnqueuer{}
		handler := NewAuditEventHandler(enqueuer)

		event := mockDomainEvent{
			action:     domain.AuditActionUpdate,
			entityType: domain.EntityTypeSettings,
			entityID:   1,
			userID:     0,
			message:    "system updated settings",
		}
		if err := handler.Handle(context.Background(), event); err != nil {
			t.Fatalf("handle: %v", err)
		}
		if len(enqueuer.payloads) != 0 {
			t.Fatalf("expected 0 enqueued payloads for user_id=0 non-login event, got %d", len(enqueuer.payloads))
		}
	})

	// Failed login with user_id=0 → kept.
	t.Run("keeps login event when user_id=0", func(t *testing.T) {
		enqueuer := &mockAuditEnqueuer{}
		handler := NewAuditEventHandler(enqueuer)

		event := domain.UserLoginEvent{
			BaseEvent: domain.BaseEvent{
				EventName:    "UserLogin",
				ActorUserID:  0,
				AuditMessage: "đăng nhập thất bại",
				Action:       domain.AuditActionLogin,
				EntityType:   domain.EntityTypeUser,
			},
			Success: false,
			Reason:  "invalid_credentials",
		}
		if err := handler.Handle(context.Background(), event); err != nil {
			t.Fatalf("handle: %v", err)
		}
		if len(enqueuer.payloads) != 1 {
			t.Fatalf("expected 1 enqueued payload for failed login (user_id=0 + LOGIN action), got %d", len(enqueuer.payloads))
		}
	})

	// Non-zero user_id → kept regardless of action.
	t.Run("keeps non-login event when user_id is set", func(t *testing.T) {
		enqueuer := &mockAuditEnqueuer{}
		handler := NewAuditEventHandler(enqueuer)

		event := mockDomainEvent{
			action:     domain.AuditActionUpdate,
			entityType: domain.EntityTypeSettings,
			entityID:   1,
			userID:     42,
			message:    "Frank Nguyen updated settings",
		}
		if err := handler.Handle(context.Background(), event); err != nil {
			t.Fatalf("handle: %v", err)
		}
		if len(enqueuer.payloads) != 1 {
			t.Fatalf("expected 1 enqueued payload for non-zero user_id event, got %d", len(enqueuer.payloads))
		}
	})
}

// TestProperty13_BulkTransferMetadataCompleteness verifies that when a
// BulkTransferResultParsedEvent is handled, the enqueued payload metadata
// contains all five required keys with matching values.
//
// Validates: Requirements 9.1
func TestProperty13_BulkTransferMetadataCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(13131)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Constrain int values to [0, 1<<53) so they round-trip through float64 without precision loss.
	genSafeInt := gen.Int().Map(func(v int) int {
		if v < 0 {
			v = -v
		}
		const maxSafeFloat64Int = 1 << 53
		return v % maxSafeFloat64Int
	})
	genSafeInt64 := gen.Int64().Map(func(v int64) int64 {
		if v < 0 {
			v = -v
		}
		const maxSafeFloat64Int64 = int64(1) << 53
		return v % maxSafeFloat64Int64
	})

	genBulkTransferEvent := gopter.CombineGens(
		genNonEmptyAlphaString(), // filename
		genSafeInt,               // total
		genSafeInt,               // completed
		genSafeInt,               // failed
		genSafeInt64,             // total_amount
	).Map(func(values []interface{}) domain.BulkTransferResultParsedEvent {
		return domain.BulkTransferResultParsedEvent{
			BaseEvent: domain.BaseEvent{
				EventName:    "BulkTransferResultParsed",
				ActorUserID:  1,
				AuditMessage: "đã xử lý chuyển khoản hàng loạt",
				Action:       domain.AuditActionBulkCreate,
				EntityType:   domain.EntityTypeTransaction,
			},
			Filename:       values[0].(string),
			TotalTransfers: values[1].(int),
			CompletedCount: values[2].(int),
			FailedCount:    values[3].(int),
			TotalAmount:    values[4].(int64),
		}
	})

	properties.Property("BulkTransfer metadata contains all five required keys with matching values", prop.ForAll(
		func(event domain.BulkTransferResultParsedEvent) bool {
			enqueuer := &mockAuditEnqueuer{}
			handler := NewAuditEventHandler(enqueuer)

			if err := handler.Handle(context.Background(), event); err != nil {
				return false
			}

			if len(enqueuer.payloads) != 1 {
				return false
			}

			p := enqueuer.payloads[0]
			if p.MetadataJSON == "" {
				return false
			}

			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(p.MetadataJSON), &meta); err != nil {
				return false
			}

			// Check "filename"
			filenameVal, ok := meta["filename"]
			if !ok || filenameVal != event.Filename {
				return false
			}

			// Check "total" — JSON numbers unmarshal as float64
			totalVal, ok := meta["total"]
			if !ok {
				return false
			}
			if int(totalVal.(float64)) != event.TotalTransfers {
				return false
			}

			// Check "completed"
			completedVal, ok := meta["completed"]
			if !ok {
				return false
			}
			if int(completedVal.(float64)) != event.CompletedCount {
				return false
			}

			// Check "failed"
			failedVal, ok := meta["failed"]
			if !ok {
				return false
			}
			if int(failedVal.(float64)) != event.FailedCount {
				return false
			}

			// Check "total_amount"
			totalAmountVal, ok := meta["total_amount"]
			if !ok {
				return false
			}
			if int64(totalAmountVal.(float64)) != event.TotalAmount {
				return false
			}

			return true
		},
		genBulkTransferEvent,
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
