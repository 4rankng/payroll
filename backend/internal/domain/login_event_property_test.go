// Feature: audit-log-revamp, Property 6: Login event carries IP and UserAgent
package domain

import (
	"context"
	"testing"

	auditctx "api-server/internal/pkg/context"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// genNonEmptyString generates a non-empty string (alpha + digits).
func genNonEmptyString() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 })
}

// TestProperty6_LoginEventCarriesIPAndUserAgent verifies that a UserLoginEvent
// constructed from a context that contains an IP address and user agent will
// expose those values via GetIPAddress() and GetUserAgent().
//
// Validates: Requirements 4.1, 4.2, 4.6
func TestProperty6_LoginEventCarriesIPAndUserAgent(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(6666)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("login event GetIPAddress and GetUserAgent equal context values", prop.ForAll(
		func(ip, ua string) bool {
			// Build a context that carries the IP address and user agent.
			ctx := context.Background()
			ctx = auditctx.WithIPAddress(ctx, ip)
			ctx = auditctx.WithUserAgent(ctx, ua)

			// Construct the login event using the enriched context.
			// Pass empty explicit ipAddress/userAgent to confirm context takes precedence.
			event := NewUserLoginEvent(ctx, 42, "testuser", "Test User", "", "", "", "")

			return event.GetIPAddress() == ip && event.GetUserAgent() == ua
		},
		genNonEmptyString(), // ip
		genNonEmptyString(), // ua
	))

	properties.Property("login event falls back to explicit params when context has no IP/UA", prop.ForAll(
		func(ip, ua string) bool {
			// Use a bare context with no audit info.
			ctx := context.Background()

			// Pass explicit ipAddress and userAgent parameters.
			event := NewUserLoginEvent(ctx, 42, "testuser", "Test User", ip, ua, "", "")

			return event.GetIPAddress() == ip && event.GetUserAgent() == ua
		},
		genNonEmptyString(), // ip
		genNonEmptyString(), // ua
	))

	properties.Property("context IP/UA takes precedence over explicit params", prop.ForAll(
		func(ctxIP, ctxUA, explicitIP, explicitUA string) bool {
			ctx := context.Background()
			ctx = auditctx.WithIPAddress(ctx, ctxIP)
			ctx = auditctx.WithUserAgent(ctx, ctxUA)

			event := NewUserLoginEvent(ctx, 42, "testuser", "Test User", explicitIP, explicitUA, "", "")

			// Context values should be used since they are non-empty.
			return event.GetIPAddress() == ctxIP && event.GetUserAgent() == ctxUA
		},
		genNonEmptyString(), // ctxIP
		genNonEmptyString(), // ctxUA
		genNonEmptyString(), // explicitIP
		genNonEmptyString(), // explicitUA
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
