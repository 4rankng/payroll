package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const flowSettlementSim = "SettlementSimulation"

// SimulateSettlementRequestV2 mirrors the current backend dto.SimulateSettlementRequest.
type SimulateSettlementRequestV2 struct {
	StartDate   string `json:"start_date"`
	ExportCount int    `json:"export_count,omitempty"`
	CadenceDays int    `json:"cadence_days,omitempty"`
}

// SimulationResultShapeV2 is a minimal subset of dto.SimulationResult for assertions.
type SimulationResultShapeV2 struct {
	Verdict     string   `json:"verdict"`
	StartDate   string   `json:"start_date"`
	ExportDates []string `json:"export_dates"`
	Summary     struct {
		TotalEligibleTimesheets int   `json:"total_eligible_timesheets"`
		TotalEligibleAmount     int64 `json:"total_eligible_amount"`
		RemainingTimesheets     int   `json:"remaining_timesheets"`
		AllSettled              bool  `json:"all_settled"`
	} `json:"summary"`
	Exports    []json.RawMessage `json:"exports"`
	Remainders []json.RawMessage `json:"remainders"`
}

// runSettlementSimulationTests exercises the read-only /payrolls/simulate-settlement
// endpoint end-to-end. Mirrors the real "Xuất sao kê" selection via
// PayrollReportByProjectService.
func runSettlementSimulationTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Settlement Simulation (Mô phỏng đối soát)")

	admin := client.WithToken(data.AdminToken)

	// ── 1. ADMIN → 200 + valid verdict ─────────────────────────────────
	reporter.RunTest(flowSettlementSim, "Admin can run simulation and gets a valid verdict", func() error {
		req := SimulateSettlementRequestV2{StartDate: "2026-07-26", ExportCount: 2, CadenceDays: 7}
		apiResp, status, err := admin.Post("/api/v1/payrolls/simulate-settlement", req)
		if err != nil {
			return fmt.Errorf("simulate-settlement: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("expected HTTP %d, got %d (msg: %s)", http.StatusOK, status, apiResp.Message)
		}

		var result SimulationResultShapeV2
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}

		if result.Verdict != "AN_TOAN_DE_XUAT" && result.Verdict != "CAN_KIEM_TRA" {
			return fmt.Errorf("verdict = %q, want AN_TOAN_DE_XUAT or CAN_KIEM_TRA", result.Verdict)
		}
		if len(result.ExportDates) != 2 {
			return fmt.Errorf("export_dates len = %d, want 2", len(result.ExportDates))
		}
		fmt.Printf("    Verdict=%s, eligible=%d ts / ₫%d, remainders=%d\n",
			result.Verdict, result.Summary.TotalEligibleTimesheets, result.Summary.TotalEligibleAmount,
			len(result.Remainders))
		return nil
	})

	// ── 2. Export count clamping (0 → 4, 99 → 10) ─────────────────────
	reporter.RunTest(flowSettlementSim, "Export count clamped to [1,10]", func() error {
		apiResp, _, err := admin.Post("/api/v1/payrolls/simulate-settlement", SimulateSettlementRequestV2{StartDate: "2026-07-26"})
		if err != nil {
			return err
		}
		var r SimulationResultShapeV2
		raw, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		if len(r.ExportDates) != 4 {
			return fmt.Errorf("export_count=0 → %d dates, want 4 (default)", len(r.ExportDates))
		}

		apiResp, _, err = admin.Post("/api/v1/payrolls/simulate-settlement", SimulateSettlementRequestV2{StartDate: "2026-07-26", ExportCount: 99})
		if err != nil {
			return err
		}
		raw, _ = json.Marshal(apiResp.Data)
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		if len(r.ExportDates) != 10 {
			return fmt.Errorf("export_count=99 → %d dates, want 10 (max clamp)", len(r.ExportDates))
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
		_, status, err := partner.Post("/api/v1/payrolls/simulate-settlement", SimulateSettlementRequestV2{StartDate: "2026-07-26"})
		if err != nil && status != http.StatusForbidden {
			return fmt.Errorf("unexpected error: %w", err)
		}
		if status != http.StatusForbidden {
			return fmt.Errorf("partner → HTTP %d, want 403", status)
		}
		return nil
	})

	// ── 4. Determinism (back-to-back same verdict) ────────────────────
	reporter.RunTest(flowSettlementSim, "Back-to-back simulations are deterministic", func() error {
		req := SimulateSettlementRequestV2{StartDate: "2026-07-26", ExportCount: 1}
		r1, _, err := admin.Post("/api/v1/payrolls/simulate-settlement", req)
		if err != nil {
			return err
		}
		r2, _, err := admin.Post("/api/v1/payrolls/simulate-settlement", req)
		if err != nil {
			return err
		}
		var s1, s2 SimulationResultShapeV2
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
		if s1.Summary.TotalEligibleTimesheets != s2.Summary.TotalEligibleTimesheets {
			return fmt.Errorf("eligible count drifted: %d vs %d", s1.Summary.TotalEligibleTimesheets, s2.Summary.TotalEligibleTimesheets)
		}
		return nil
	})
}
