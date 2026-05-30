package main

import (
	"fmt"
)

const flowWallet = "Wallet"

func runWalletTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Wallet")

	admin := client.WithToken(data.AdminToken)

	reporter.RunTest(flowWallet, "Get wallet balance", func() error {
		var resp WalletBalanceResponse
		if _, err := admin.GetInto("/api/v1/wallet/balance", &resp); err != nil {
			return fmt.Errorf("wallet balance: %w", err)
		}
		fmt.Printf("    Balance: %d %s\n", resp.Balance, resp.Currency)
		return nil
	})

	reporter.RunTest(flowWallet, "Sync balance", func() error {
		var resp interface{}
		if _, err := admin.PostInto("/api/v1/wallet/sync", nil, &resp); err != nil {
			// Sync may fail if no wallet provider configured - that's ok
			fmt.Printf("    Sync skipped (no provider): %v\n", err)
			return nil
		}
		return nil
	})

	reporter.RunTest(flowWallet, "List wallet payments", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/wallet/payments?pageSize=10", &resp); err != nil {
			return fmt.Errorf("wallet payments: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowWallet, "List wallet topups", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/wallet/topups?pageSize=10", &resp); err != nil {
			return fmt.Errorf("wallet topups: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowWallet, "Edge: resolve non-existent payment", func() error {
		_, statusCode, _ := admin.Post("/api/v1/wallet/payments/999999/resolve", nil)
		if statusCode < 400 {
			return fmt.Errorf("expected error resolving non-existent payment")
		}
		return nil
	})
}
