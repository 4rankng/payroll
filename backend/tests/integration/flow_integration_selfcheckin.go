package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
)

const flowIntegrationSelfCheckin = "IntegrationSelfCheckin"

// selfCheckinSupportedCode is the project code the chatbot self check-in flow
// is configured for by default (SELF_CHECKIN_SUPPORTED_PROJECT_CODES, env
// overridable). Live-data assertions that need such a project skip when the
// dev DB has none.
const selfCheckinSupportedCode = "LGD"

// msgSelfCheckinProjectNotSupported mirrors the server's config-gate message
// so the flow test pins the exact text the bot relays.
const msgSelfCheckinProjectNotSupported = "Dự án này chưa hỗ trợ bật/tắt tự chấm công qua chatbot."

// selfCheckinUpdateRequest is the body for /integration/self-checkin/update.
type selfCheckinUpdateRequest struct {
	ActionToken string `json:"action_token"`
	ProjectID   uint   `json:"project_id"`
	Enable      *bool  `json:"enable"`
}

// selfCheckinUpdateResponse is the verdict payload. EffectiveFrom is a plain
// string: it is omitted by the server when a queued enable was cancelled.
type selfCheckinUpdateResponse struct {
	Success                 bool   `json:"success"`
	Kind                    string `json:"kind"`
	Immediate               bool   `json:"immediate"`
	EffectiveFrom           string `json:"effective_from"`
	CancelledPendingEnable  bool   `json:"cancelled_pending_enable"`
	CancelledPendingDisable bool   `json:"cancelled_pending_disable"`
}

// uniqueActionToken returns a random 256-bit hex token, the same entropy class
// the server mints for verified tokens.
func uniqueActionToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// seedVerifiedToken writes a verified action token directly into one flow's
// Redis namespace ("zsc" or "zreset"), mirroring what a successful verify
// mints. The OTP round-trip cannot be driven cross-process — the code exists
// only inside the server process and is stored hashed — so the update and
// reset steps are seeded past it, exactly like queryAdvancePaymentUploadDates
// seeds past HTTP-only surfaces. Returns the token plus a best-effort cleanup.
func seedVerifiedToken(namespace, uid string) (string, func(), error) {
	token, err := uniqueActionToken()
	if err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256([]byte(token))
	key := namespace + "-ok:" + hex.EncodeToString(sum[:])
	out, err := execCommandInContainer("payroll-redis", "redis-cli", "SET", key, uid, "EX", "300")
	if err != nil || out != "OK" {
		return "", nil, fmt.Errorf("seed %s (uid %s): %v (%q)", key, uid, err, out)
	}
	cleanup := func() {
		_, _ = execCommandInContainer("payroll-redis", "redis-cli", "DEL", key)
	}
	return token, cleanup, nil
}

// execCommandInContainer runs a command inside a dev compose container and
// returns trimmed combined output.
func execCommandInContainer(container string, args ...string) (string, error) {
	out, err := exec.Command("docker", append([]string{"exec", container}, args...)...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// runIntegrationSelfCheckinTests exercises the chatbot self check-in channel:
// the 401 guards, the found=false shape, the supported-project gate, and —
// via a seeded verified token, since the OTP itself cannot leave the server
// process — the full update verdict including cross-flow replay isolation.
func runIntegrationSelfCheckinTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Integration Self Check-in (chatbot enable/disable)")

	admin := client.WithToken(data.AdminToken)

	var keyID uint
	var key string
	reporter.RunTest(flowIntegrationSelfCheckin, "Admin creates an API key for the flow", func() error {
		var out CreateAPIKeyResponse
		if _, err := admin.PostInto("/api/v1/admin/api-keys", CreateAPIKeyRequest{Name: "itest self-checkin"}, &out); err != nil {
			return err
		}
		if out.Key == "" {
			return fmt.Errorf("create returned an empty plaintext key")
		}
		keyID, key = out.ID, out.Key
		return nil
	})
	defer func() {
		if keyID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/admin/api-keys/%d", keyID))
		}
	}()

	// The supported/unsupported pair comes from live discovery so the gate is
	// proven against real project ids.
	var supportedProjectID, unsupportedProjectID uint
	for _, p := range data.Projects {
		if p.Code == selfCheckinSupportedCode {
			if supportedProjectID == 0 {
				supportedProjectID = p.ID
			}
		} else if unsupportedProjectID == 0 {
			unsupportedProjectID = p.ID
		}
	}

	anonBody := IntegrationOTPRequest{Phone: "0987654321"}
	reporter.RunTest(flowIntegrationSelfCheckin, "OTP without X-API-Key returns 401", func() error {
		anon := NewAPIClient(cfg.BaseURL)
		_, code, err := anon.PostExpectError("/api/v1/integration/self-checkin/otp", anonBody)
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationSelfCheckin, "Verify without X-API-Key returns 401", func() error {
		anon := NewAPIClient(cfg.BaseURL)
		_, code, err := anon.PostExpectError("/api/v1/integration/self-checkin/verify", map[string]string{
			"session_id": "x", "code": "123456",
		})
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationSelfCheckin, "Update without X-API-Key returns 401", func() error {
		anon := NewAPIClient(cfg.BaseURL)
		_, code, err := anon.PostExpectError("/api/v1/integration/self-checkin/update", map[string]any{
			"action_token": "x", "project_id": 1, "enable": false,
		})
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	if key == "" {
		reporter.Skip(flowIntegrationSelfCheckin, "Authenticated self check-in calls", "API key creation failed")
		return
	}
	machine := NewAPIClient(cfg.BaseURL)
	machine.Headers = map[string]string{"X-API-Key": key}

	reporter.RunTest(flowIntegrationSelfCheckin, "OTP with garbage key returns 401", func() error {
		bad := NewAPIClient(cfg.BaseURL)
		bad.Headers = map[string]string{"X-API-Key": "ttk_garbage"}
		_, code, err := bad.PostExpectError("/api/v1/integration/self-checkin/otp", anonBody)
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationSelfCheckin, "OTP for unknown phone reports found=false", func() error {
		var out IntegrationOTPResponse
		if _, err := machine.PostInto("/api/v1/integration/self-checkin/otp", IntegrationOTPRequest{Phone: uniqueMobile()}, &out); err != nil {
			return err
		}
		if out.Found || out.OTPSent {
			return fmt.Errorf("found=%v otp_sent=%v, want false/false", out.Found, out.OTPSent)
		}
		if out.FailureReason == nil || *out.FailureReason != "account_not_found" {
			return fmt.Errorf("failure_reason = %v, want account_not_found", out.FailureReason)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationSelfCheckin, "OTP with a malformed phone returns 400", func() error {
		_, code, err := machine.PostExpectError("/api/v1/integration/self-checkin/otp", IntegrationOTPRequest{Phone: "034090005032"})
		if err != nil {
			return err
		}
		if code != 400 {
			return fmt.Errorf("status %d, want 400", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationSelfCheckin, "Verify with a garbage session returns 401 or 429", func() error {
		_, code, err := machine.PostExpectError("/api/v1/integration/self-checkin/verify", map[string]string{
			"session_id": "garbage-session", "code": "123456",
		})
		if err != nil {
			return err
		}
		if code != 401 && code != 429 {
			return fmt.Errorf("status %d, want 401 or 429", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationSelfCheckin, "Update for an unsupported project is rejected before the token is consumed", func() error {
		probe := unsupportedProjectID
		if probe == 0 {
			probe = 4_000_000_000 // no other project exists: a huge id gets the same rejection
		}
		enabled := false
		apiErr, code, err := machine.PostExpectError("/api/v1/integration/self-checkin/update", selfCheckinUpdateRequest{
			ActionToken: "garbage-token", ProjectID: probe, Enable: &enabled,
		})
		if err != nil {
			return err
		}
		if code != 400 {
			return fmt.Errorf("status %d, want 400", code)
		}
		if apiErr.Message != msgSelfCheckinProjectNotSupported {
			return fmt.Errorf("message = %q, want the exact config-gate text", apiErr.Message)
		}
		return nil
	})

	if supportedProjectID == 0 {
		reporter.Skip(flowIntegrationSelfCheckin, "Update with garbage token on a supported project",
			fmt.Sprintf("no project with code %s in dev data", selfCheckinSupportedCode))
	} else {
		reporter.RunTest(flowIntegrationSelfCheckin, "Update with garbage token on a supported project dies on the token (401/429)", func() error {
			enabled := false
			_, code, err := machine.PostExpectError("/api/v1/integration/self-checkin/update", selfCheckinUpdateRequest{
				ActionToken: "garbage-token", ProjectID: supportedProjectID, Enable: &enabled,
			})
			if err != nil {
				return err
			}
			if code != 401 && code != 429 {
				return fmt.Errorf("status %d, want 401 or 429", code)
			}
			return nil
		})
	}

	// --- Cross-flow replay isolation, proven over the wire -------------------
	//
	// The token is seeded into one flow's namespace by hand; the OTHER flow's
	// endpoint must reject it. On the reset side a rejection is non-mutating;
	// a 2xx here would mean the namespaces merged and a self check-in token
	// can set passwords — the exact security regression this pins.
	uid, uidErr := queryPayrollScalar("SELECT id FROM users WHERE deleted_at IS NULL ORDER BY id LIMIT 1")
	if uidErr != nil || uid == "" {
		reporter.Skip(flowIntegrationSelfCheckin, "Cross-flow replay isolation", "cannot read a user id from the dev DB")
		return
	}

	reporter.RunTest(flowIntegrationSelfCheckin, "Seeded self check-in token is dead at /password-reset/reset", func() error {
		token, cleanup, err := seedVerifiedToken("zsc", uid)
		if err != nil {
			return err
		}
		defer cleanup()
		_, code, err := machine.PostExpectError("/api/v1/integration/password-reset/reset", map[string]string{
			"reset_token":  token,
			"new_password": "SelfCheckinReplay1!",
		})
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401 (namespaces merged?)", code)
		}
		return nil
	})

	reporter.RunTest(flowIntegrationSelfCheckin, "Seeded reset token is dead at self check-in update", func() error {
		if supportedProjectID == 0 {
			reporter.Skip(flowIntegrationSelfCheckin, "Seeded reset token is dead at self check-in update",
				fmt.Sprintf("no project with code %s in dev data", selfCheckinSupportedCode))
			return nil
		}
		token, cleanup, err := seedVerifiedToken("zreset", uid)
		if err != nil {
			return err
		}
		defer cleanup()
		enabled := false
		_, code, err := machine.PostExpectError("/api/v1/integration/self-checkin/update", selfCheckinUpdateRequest{
			ActionToken: token, ProjectID: supportedProjectID, Enable: &enabled,
		})
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401 (namespaces merged?)", code)
		}
		return nil
	})

	// --- Seeded update: the full verdict path --------------------------------
	if supportedProjectID == 0 {
		reporter.Skip(flowIntegrationSelfCheckin, "Seeded update verdict",
			fmt.Sprintf("no project with code %s in dev data", selfCheckinSupportedCode))
		return
	}
	runSeededSelfCheckinUpdate(machine, reporter, supportedProjectID)

	reporter.RunTest(flowIntegrationSelfCheckin, "Revoked key can no longer authenticate", func() error {
		if keyID == 0 {
			return fmt.Errorf("no key created")
		}
		if _, code, err := admin.Delete(fmt.Sprintf("/api/v1/admin/api-keys/%d", keyID)); err != nil {
			return err
		} else if code != 200 {
			return fmt.Errorf("revoke status %d, want 200", code)
		}
		keyID = 0 // the deferred cleanup would otherwise revoke a second time
		_, code, err := machine.PostExpectError("/api/v1/integration/self-checkin/otp", anonBody)
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("post-revoke status %d, want 401", code)
		}
		return nil
	})
}

// runSeededSelfCheckinUpdate drives one real enable/disable verdict against
// the live API with a seeded verified token, then restores the assignment row
// it touched. Skips when the dev DB has no eligible active assignment with a
// linked user, or when an enable is refused by project config (payrate/quota)
// — that refusal belongs to the unit-test matrix.
func runSeededSelfCheckinUpdate(machine *APIClient, reporter *Reporter, projectID uint) {
	testName := "Seeded update returns the verdict payload (single use)"
	row, rowErr := queryPayrollScalar(fmt.Sprintf(
		"SELECT CONCAT_WS('|', pe.id, e.user_id, pe.check_in_enabled,"+
			" IFNULL(pe.pending_check_in_enabled,'NULL'),"+
			" IFNULL(DATE_FORMAT(pe.check_in_effective_from,'%%Y-%%m-%%d'),'NULL'),"+
			" IFNULL(DATE_FORMAT(pe.check_in_start_date,'%%Y-%%m-%%d'),'NULL')) "+
			"FROM project_employees pe JOIN employees e ON e.id = pe.employee_id "+
			"WHERE pe.project_id = %d AND pe.last_date IS NULL AND pe.deleted_at IS NULL"+
			" AND e.user_id IS NOT NULL AND e.deleted_at IS NULL ORDER BY pe.id LIMIT 1", projectID))
	if rowErr != nil || row == "" {
		reporter.Skip(flowIntegrationSelfCheckin, testName, "no eligible active assignment with a linked user in dev data")
		return
	}
	parts := strings.Split(row, "|")
	if len(parts) != 6 {
		reporter.Skip(flowIntegrationSelfCheckin, testName, "no eligible active assignment with a linked user in dev data")
		return
	}
	peID, uidStr, enabledStr, pendingStr, effectiveFromStr, startDateStr := parts[0], parts[1], parts[2], parts[3], parts[4], parts[5]

	restore := func() {
		_ = execPayrollSQL(fmt.Sprintf(
			"UPDATE project_employees SET check_in_enabled=%s, pending_check_in_enabled=%s,"+
				" check_in_effective_from=%s, check_in_start_date=%s WHERE id=%s",
			enabledStr, sqlOrNull(pendingStr), sqlOrNull(effectiveFromStr), sqlOrNull(startDateStr), peID))
	}
	defer restore()

	if enabledStr != "1" {
		payrates, payrateErr := queryPayrollScalar(fmt.Sprintf(
			"SELECT COUNT(*) FROM payrates WHERE project_id = %d AND deleted_at IS NULL", projectID))
		if payrateErr == nil && payrates == "0" {
			reporter.Skip(flowIntegrationSelfCheckin, testName, "project has no payrate config: enable would refuse in this env")
			return
		}
	}
	enable := enabledStr != "1"

	reporter.RunTest(flowIntegrationSelfCheckin, testName, func() error {
		token, cleanup, err := seedVerifiedToken("zsc", uidStr)
		if err != nil {
			return err
		}
		defer cleanup()

		var out selfCheckinUpdateResponse
		if _, err := machine.PostInto("/api/v1/integration/self-checkin/update", selfCheckinUpdateRequest{
			ActionToken: token, ProjectID: projectID, Enable: &enable,
		}, &out); err != nil {
			if enable {
				reporter.Skip(flowIntegrationSelfCheckin, testName, "enable refused by project config: "+err.Error())
				return nil
			}
			return err
		}

		wantKind := "disable"
		if enable {
			wantKind = "enable"
		}
		if !out.Success || out.Kind != wantKind {
			return fmt.Errorf("success=%v kind=%q, want true/%q", out.Success, out.Kind, wantKind)
		}
		if out.EffectiveFrom != "" && !strings.HasSuffix(strings.Split(out.EffectiveFrom, "T")[0], "-01") {
			return fmt.Errorf("effective_from %q is not a day-1", out.EffectiveFrom)
		}
		if out.EffectiveFrom == "" && !out.CancelledPendingEnable && !out.CancelledPendingDisable {
			return fmt.Errorf("no effective_from and no cancellation flag: %+v", out)
		}

		// The action token is single use at the HTTP boundary too.
		_, code, err := machine.PostExpectError("/api/v1/integration/self-checkin/update", selfCheckinUpdateRequest{
			ActionToken: token, ProjectID: projectID, Enable: &enable,
		})
		if err != nil {
			return err
		}
		if code != 401 && code != 429 {
			return fmt.Errorf("replay status %d, want 401 or 429", code)
		}
		return nil
	})
}

// sqlOrNull renders a snapshotted DB value back into SQL: bare NULL or a
// quoted literal.
func sqlOrNull(v string) string {
	if v == "" || v == "NULL" {
		return "NULL"
	}
	return "'" + v + "'"
}
