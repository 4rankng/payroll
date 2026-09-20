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

	reporter.RunTest(flowWallet, "Get current-state demand forecast", func() error {
		var resp WalletDemandForecastResponse
		if _, err := admin.GetInto("/api/v1/wallet/demand-forecast", &resp); err != nil {
			return fmt.Errorf("wallet demand forecast: %w", err)
		}
		if resp.CurrentForMonth == "" || resp.MaxCycleDay <= 0 || len(resp.Periods) == 0 {
			return fmt.Errorf(
				"incomplete forecast response: forMonth=%q maxCycleDay=%d periods=%d",
				resp.CurrentForMonth,
				resp.MaxCycleDay,
				len(resp.Periods),
			)
		}
		if resp.Prediction.RecommendedBalance < 0 ||
			resp.Prediction.P50Reference > resp.Prediction.P90Reference ||
			resp.Prediction.P90Reference > resp.Prediction.P99Reference {
			return fmt.Errorf(
				"invalid forecast ladder: recommended=%d p50=%d p90=%d p99=%d",
				resp.Prediction.RecommendedBalance,
				resp.Prediction.P50Reference,
				resp.Prediction.P90Reference,
				resp.Prediction.P99Reference,
			)
		}
		return nil
	})

	// Sync pulls the balance from the active provider, so it can only succeed
	// where one is registered; elsewhere the API answers 400 by design.
	if hasRegisteredDisbursementProvider(admin) {
		reporter.RunTest(flowWallet, "Sync balance", func() error {
			var resp interface{}
			if _, err := admin.PostInto("/api/v1/wallet/balance/sync", nil, &resp); err != nil {
				return fmt.Errorf("sync wallet balance: %w", err)
			}
			return nil
		})
	} else {
		reporter.Skip(flowWallet, "Sync balance", "no disbursement provider registered in this deployment")
	}

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
		_, statusCode, _ := admin.Post(fmt.Sprintf("/api/v1/wallet/payments/%d/resolve", nonexistentID), nil)
		if statusCode < 400 {
			return fmt.Errorf("expected error resolving non-existent payment")
		}
		return nil
	})
}
