// Feature: audit-log-revamp, Property 11: Audit message contains entity-specific information
package domain

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// genNonEmptyAlphaString generates a random non-empty alphabetic string.
func genNonEmptyAlphaString() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0
	})
}

// TestProperty11_BankAuditMessageContainsBranchName verifies that NewBankCreatedEvent
// produces an AuditMessage containing the bank's BranchName, with Action=CREATE and
// EntityType=bank.
//
// Validates: Requirements 6.2, 3.12
func TestProperty11_BankAuditMessageContainsBranchName(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(11001)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("BankCreatedEvent AuditMessage contains BranchName", prop.ForAll(
		func(branchName string) bool {
			bank := &Bank{
				ID:         1,
				BranchName: branchName,
			}
			event := NewBankCreatedEvent(context.Background(), bank)

			msg := event.GetAuditMessage()
			return msg != "" &&
				strings.Contains(msg, branchName) &&
				event.GetAction() == AuditActionCreate &&
				event.GetEntityType() == EntityTypeBank
		},
		genNonEmptyAlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_PayrateAuditMessageContainsProjectName verifies that NewPayrateCreatedEvent
// produces an AuditMessage containing the project name, with Action=CREATE and
// EntityType=payrate.
//
// Validates: Requirements 6.3, 3.12
func TestProperty11_PayrateAuditMessageContainsProjectName(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(11002)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("PayrateCreatedEvent AuditMessage contains project name", prop.ForAll(
		func(projectName string) bool {
			payrate := &Payrate{
				ID:        1,
				ProjectID: 1,
				FromDate:  time.Now(),
				Project: Project{
					ID:   1,
					Name: projectName,
				},
			}
			event := NewPayrateCreatedEvent(context.Background(), payrate)

			msg := event.GetAuditMessage()
			return msg != "" &&
				strings.Contains(msg, projectName) &&
				event.GetAction() == AuditActionCreate &&
				event.GetEntityType() == EntityTypePayrate
		},
		genNonEmptyAlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_AssetAuditMessageContainsFilename verifies that NewAssetCreatedEvent
// produces an AuditMessage containing the asset's Filename, with Action=CREATE and
// EntityType=asset.
//
// Validates: Requirements 6.7, 3.12
func TestProperty11_AssetAuditMessageContainsFilename(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(11003)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("AssetCreatedEvent AuditMessage contains Filename", prop.ForAll(
		func(filename string) bool {
			asset := &Asset{
				ID:       1,
				Filename: filename,
				FilePath: "/uploads/" + filename,
			}
			event := NewAssetCreatedEvent(context.Background(), asset)

			msg := event.GetAuditMessage()
			return msg != "" &&
				strings.Contains(msg, filename) &&
				event.GetAction() == AuditActionCreate &&
				event.GetEntityType() == EntityTypeAsset
		},
		genNonEmptyAlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_LenderAuditMessageContainsName verifies that NewLenderCreatedEvent
// produces an AuditMessage containing the lender's Name, with Action=CREATE and
// EntityType=lender.
//
// Validates: Requirements 6.8, 3.12
func TestProperty11_LenderAuditMessageContainsName(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(11004)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("LenderCreatedEvent AuditMessage contains lender Name", prop.ForAll(
		func(lenderName string) bool {
			lender := &Lender{
				ID:   1,
				Name: lenderName,
			}
			event := NewLenderCreatedEvent(context.Background(), lender)

			msg := event.GetAuditMessage()
			return msg != "" &&
				strings.Contains(msg, lenderName) &&
				event.GetAction() == AuditActionCreate &&
				event.GetEntityType() == EntityTypeLender
		},
		genNonEmptyAlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_SettingsAuditMessageContainsKey verifies that NewSettingsCreatedEvent
// produces an AuditMessage containing the settings key, with Action=CREATE and
// EntityType=settings.
//
// Validates: Requirements 6.10, 3.12
func TestProperty11_SettingsAuditMessageContainsKey(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(11005)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("SettingsCreatedEvent AuditMessage contains settings key", prop.ForAll(
		func(key string) bool {
			event := NewSettingsCreatedEvent(context.Background(), key, 1)

			msg := event.GetAuditMessage()
			return msg != "" &&
				strings.Contains(msg, key) &&
				event.GetAction() == AuditActionCreate &&
				event.GetEntityType() == EntityTypeSettings
		},
		genNonEmptyAlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty11_LedgerEntryAuditMessageContainsAccount verifies that NewLedgerEntryCreatedEvent
// produces an AuditMessage containing the account name, with Action=CREATE and
// EntityType=ledger_entry.
//
// Validates: Requirements 6.9, 3.12
func TestProperty11_LedgerEntryAuditMessageContainsAccount(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(11006)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Use valid LedgerAccount values since the factory uses string(entry.Account) directly
	validAccounts := GetValidLedgerAccounts()

	properties.Property("LedgerEntryCreatedEvent AuditMessage contains account name", prop.ForAll(
		func(idx uint) bool {
			account := validAccounts[int(idx)%len(validAccounts)]
			entry := &LedgerEntry{
				ID:      1,
				Account: account,
				Debit:   10000,
				Credit:  0,
				Date:    time.Now(),
				Party:   "test-party",
			}
			event := NewLedgerEntryCreatedEvent(context.Background(), entry)

			msg := event.GetAuditMessage()
			return msg != "" &&
				strings.Contains(msg, account) &&
				event.GetAction() == AuditActionCreate &&
				event.GetEntityType() == EntityTypeLedgerEntry
		},
		gen.UInt(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
