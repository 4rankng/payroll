// Feature: audit-log-revamp, Property 12: Actor identity propagation
package domain

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	auditctx "api-server/internal/pkg/context"
)

// genNonZeroUint generates a random non-zero uint suitable for use as a user ID.
func genNonZeroUint() gopter.Gen {
	return gen.UInt().Map(func(v uint) uint {
		if v == 0 {
			return 1
		}
		return v
	})
}

// genUserWithID generates a minimal *User with the given ID for use in factory calls.
func genUserWithID(userID uint) *User {
	return &User{
		ID:       userID,
		Username: "testuser",
		Fullname: "Test User",
		Role:     RoleAdmin,
	}
}

// TestProperty12_ActorIdentityFromContext verifies that when a user ID is set in context,
// the resulting event's ActorUserID equals the context user ID.
//
// Validates: Requirements 8.1
func TestProperty12_ActorIdentityFromContext(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(12001)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("ActorUserID equals context user ID when context has user ID set", prop.ForAll(
		func(contextUserID uint, explicitActorID uint) bool {
			// Build a context with the contextUserID set
			ctx := auditctx.WithUserID(context.Background(), contextUserID)

			user := genUserWithID(contextUserID)

			// Call a factory that uses newBaseEventWithActor
			event := NewUserCreatedEvent(ctx, user, explicitActorID, "Test Actor")

			// The event's ActorUserID must equal the context user ID
			return event.UserID() == contextUserID
		},
		genNonZeroUint(),
		genNonZeroUint(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_ActorIdentityFallbackToExplicit verifies that when context has no user ID,
// the explicitly supplied actorUserID is used as the event's ActorUserID.
//
// Validates: Requirements 8.2
func TestProperty12_ActorIdentityFallbackToExplicit(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(12002)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("ActorUserID equals explicit actorUserID when context has no user ID", prop.ForAll(
		func(explicitActorID uint) bool {
			// Use a plain background context — no user ID set
			ctx := context.Background()

			user := genUserWithID(1)

			// Call a factory that uses newBaseEventWithActor; pass the explicit actor ID
			event := NewUserCreatedEvent(ctx, user, explicitActorID, "Test Actor")

			// The event's ActorUserID must equal the explicitly supplied actorUserID
			return event.UserID() == explicitActorID
		},
		genNonZeroUint(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_ActorIdentityContextTakesPrecedence verifies that when both context
// and explicit actorUserID are provided, the context user ID takes precedence.
//
// Validates: Requirements 8.1, 8.2
func TestProperty12_ActorIdentityContextTakesPrecedence(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(12003)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("context user ID takes precedence over explicit actorUserID", prop.ForAll(
		func(contextUserID uint, explicitActorID uint) bool {
			// Ensure the two IDs are different so we can distinguish which was used
			if contextUserID == explicitActorID {
				explicitActorID = contextUserID + 1
			}

			ctx := auditctx.WithUserID(context.Background(), contextUserID)
			user := genUserWithID(contextUserID)

			event := NewUserUpdatedEvent(ctx, user, explicitActorID, "Test Actor", nil)

			// Context user ID must win
			return event.UserID() == contextUserID
		},
		genNonZeroUint(),
		genNonZeroUint(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty12_ActorIdentityWithNewBaseEventWithActor directly tests the
// newBaseEventWithActor fallback path using NewUserUpdatedEvent with no context user ID.
//
// Validates: Requirements 8.2
func TestProperty12_ActorIdentityWithNewBaseEventWithActor(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(12004)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("explicit actorUserID is used when context has no user ID (NewUserUpdatedEvent)", prop.ForAll(
		func(explicitActorID uint) bool {
			ctx := context.Background() // no user ID in context

			user := genUserWithID(1)

			event := NewUserUpdatedEvent(ctx, user, explicitActorID, "Test Actor", nil)

			return event.UserID() == explicitActorID
		},
		genNonZeroUint(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
