package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const flowSettlementSim = "SettlementSimulation"

// SimulateSettlementRequest mirrors backend dto.SimulateSettlementRequest.
type SimulateSettlementRequest struct {
	ProjectIDs          []uint `json:"project_ids,omitempty"`
	EmployeeIDs         []uint `json:"employee_ids,omitempty"`
	ProjectedCycleCount int    `json:"projected_cycle_count,omitempty"`
}

// SimulationResultShape is a minimal subset of dto.SimulationResult — only
// the fields the integration test asserts on. Unmarshalling into a full-shape
// struct isn't worth the maintenance burden; the unit tests cover the shape.
type SimulationResultShape struct {
	Verdict             string `json:"verdict"`
	SnapshotEpoch       string `json:"snapshot_epoch"`
	ProjectedCycleCount int    `json:"projected_cycle_count"`
	Summary             struct {
		TotalEligibleCount      int   `json:"total_eligible_count"`
		TotalEligibleAmount     int64 `json:"total_eligible_amount"`
		TotalIncludedCount      int   `json:"total_included_count"`
		TotalIncludedAmount     int64 `json:"total_included_amount"`
		RemainingAfterAllCount  int   `json:"remaining_after_all_count"`
		RemainingAfterAllAmount int64 `json:"remaining_after_all_amount"`
		AllSettled              bool  `json:"all_settled"`
	} `json:"summary"`
	Reconciliation struct {
		Reconciled bool  `json:"reconciled"`
		Delta      int64 `json:"delta"`
	} `json:"reconciliation"`
	Cycles     []json.RawMessage `json:"cycles"`
	Remainders []json.RawMessage `json:"remainders"`
	Warnings   []json.RawMessage `json:"warnings"`
}

// runSettlementSimulationTests exercises the read-only /payrolls/simulate-settlement
// endpoint end-to-end. Covers:
//   - admin → 200 with a valid verdict
//   - partner → 403 (admin-only enforcement)
//   - read-only: snapshot counts unchanged across the call
//
// The full parity + straggler + stale-409 scenarios require deterministic
// timesheet seeding that's heavier than the shared test fixture supports;
// those invariants are covered by the unit suite (simulation_service_test.go,
// planner_test.go, simulation_readonly_assert_test.go) which drives the
// service directly with fakes. This flow focuses on HTTP wiring + auth +
// the no-mutation promise at the API boundary.
func runSettlementSimulationTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Settlement Simulation (Mô phỏng đối soát)")

	admin := client.WithToken(data.AdminToken)

	// ── 1. ADMIN → 200 + valid verdict ─────────────────────────────────
	reporter.RunTest(flowSettlementSim, "Admin can run simulation and gets a valid verdict", func() error {
		req := SimulateSettlementRequest{ProjectedCycleCount: 2}
		apiResp, status, err := admin.Post("/api/v1/payrolls/simulate-settlement", req)
		if err != nil {
			return fmt.Errorf("simulate-settlement: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("expected HTTP %d, got %d (msg: %s)", http.StatusOK, status, apiResp.Message)
		}
		if apiResp.Status != "success" {
			return fmt.Errorf("expected status=success, got %q", apiResp.Status)
		}

		var result SimulationResultShape
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}

		// Verdict must be one of the two payroll-scope values.
		if result.Verdict != "AN_TOAN_DE_XUAT" && result.Verdict != "CAN_KIEM_TRA" {
			return fmt.Errorf("verdict = %q, want AN_TOAN_DE_XUAT or CAN_KIEM_TRA", result.Verdict)
		}
		if result.ProjectedCycleCount != 2 {
			return fmt.Errorf("projected_cycle_count = %d, want 2", result.ProjectedCycleCount)
		}
		if len(result.Cycles) != 2 {
			return fmt.Errorf("len(cycles) = %d, want 2", len(result.Cycles))
		}
		if result.SnapshotEpoch == "" {
			return fmt.Errorf("snapshot_epoch is empty")
		}
		fmt.Printf("    Verdict=%s, eligible=%d (₫%d), included=%d, remainders=%d\n",
			result.Verdict,
			result.Summary.TotalEligibleCount, result.Summary.TotalEligibleAmount,
			result.Summary.TotalIncludedCount,
			len(result.Remainders))
		return nil
	})

	// ── 2. CYCLE COUNT CLAMPING (0 → 4, 99 → 6) ───────────────────────
	reporter.RunTest(flowSettlementSim, "Cycle count clamped to [1,6]", func() error {
		// cycleCount=0 → server clamps to default 4.
		apiResp, _, err := admin.Post("/api/v1/payrolls/simulate-settlement", SimulateSettlementRequest{ProjectedCycleCount: 0})
		if err != nil {
			return fmt.Errorf("cycleCount=0: %w", err)
		}
		var r SimulationResultShape
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		if r.ProjectedCycleCount != 4 {
			return fmt.Errorf("cycleCount=0 → %d, want 4 (default)", r.ProjectedCycleCount)
		}

		// cycleCount=99 → clamps to 6.
		apiResp, _, err = admin.Post("/api/v1/payrolls/simulate-settlement", SimulateSettlementRequest{ProjectedCycleCount: 99})
		if err != nil {
			return fmt.Errorf("cycleCount=99: %w", err)
		}
		raw, _ = json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		if r.ProjectedCycleCount != 6 {
			return fmt.Errorf("cycleCount=99 → %d, want 6 (max clamp)", r.ProjectedCycleCount)
		}
		return nil
	})

	// ── 3. PARTNER → 403 ──────────────────────────────────────────────
	reporter.RunTest(flowSettlementSim, "Partner token rejected (403)", func() error {
		if len(data.Partners) == 0 {
			reporter.Skip(flowSettlementSim, "Partner token rejected (403)", "no partner fixture")
			return nil
		}
		partner := client.WithToken(data.Partners[0].Token)
		_, status, err := partner.Post("/api/v1/payrolls/simulate-settlement", SimulateSettlementRequest{ProjectedCycleCount: 4})
		if err != nil && status != http.StatusForbidden {
			return fmt.Errorf("unexpected error: %w", err)
		}
		if status != http.StatusForbidden {
			return fmt.Errorf("partner → HTTP %d, want 403", status)
		}
		return nil
	})

	// ── 4. READ-ONLY: same verdict on back-to-back calls ──────────────
	// The full row-count snapshot requires DB access from the test runner
	// (the API doesn't expose counts). The canonical no-mutation proof is
	// the unit test simulation_readonly_assert_test.go; this HTTP-level
	// test verifies determinism (same data → same verdict, no status flips).
	reporter.RunTest(flowSettlementSim, "Back-to-back simulations are deterministic", func() error {
		req := SimulateSettlementRequest{ProjectedCycleCount: 1}
		r1, _, err := admin.Post("/api/v1/payrolls/simulate-settlement", req)
		if err != nil {
			return err
		}
		r2, _, err := admin.Post("/api/v1/payrolls/simulate-settlement", req)
		if err != nil {
			return err
		}
		var s1, s2 SimulationResultShape
		raw1, _ := json.Marshal(r1.Data)
		raw2, _ := json.Marshal(r2.Data)
		if err := json.Unmarshal(raw1, &s1); err != nil {
			return err
		}
		if err := json.Unmarshal(raw2, &s2); err != nil {
			return err
		}
		if s1.Verdict != s2.Verdict {
			return fmt.Errorf("verdict drifted: %q vs %q", s1.Verdict, s2.Verdict)
		}
		if s1.Summary.TotalEligibleCount != s2.Summary.TotalEligibleCount {
			return fmt.Errorf("eligible count drifted: %d vs %d", s1.Summary.TotalEligibleCount, s2.Summary.TotalEligibleCount)
		}
		return nil
	})
}
