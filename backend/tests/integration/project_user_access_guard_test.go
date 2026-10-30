package main

import (
	"os"
	"testing"
)

// TestProjectUserAccessGuards runs the PAYR-30 HTTP guard matrix
// (flow_project_user_access.go) against a live backend. It is skipped by
// default so `go test ./...` stays hermetic; set PAYROLL_GUARD_IT=1 with the
// API running (same contract as `make api-test`) to execute it.
func TestProjectUserAccessGuards(t *testing.T) {
	if os.Getenv("PAYROLL_GUARD_IT") == "" {
		t.Skip("set PAYROLL_GUARD_IT=1 to run against a live backend")
	}

	cfg := LoadTestConfig()
	client := NewAPIClient(cfg.BaseURL)
	if _, err := client.Login(cfg.AdminUsername, cfg.AdminPassword); err != nil {
		t.Fatalf("admin login: %v", err)
	}

	fixture, teardown, err := setupProjectUserAccessFixture(client, client.Token, cfg.UniquePrefix())
	if err != nil {
		t.Fatalf("setup fixture: %v", err)
	}
	defer teardown()

	for _, step := range puaSteps {
		step := step
		t.Run(step.name, func(t *testing.T) {
			if err := step.run(fixture); err != nil {
				t.Fatal(err)
			}
		})
	}
}
