package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/altcha-org/altcha-lib-go/v2"
	"github.com/redis/go-redis/v9"
)

// solve encodes the test helper for the widget's client-side work: brute-force
// the counter and return the base64 payload a browser would submit.
func solve(t *testing.T, challenge altcha.Challenge) string {
	t.Helper()
	solution, err := altcha.SolveChallenge(altcha.SolveChallengeOptions{
		Challenge: challenge,
		DeriveKey: altcha.DeriveKeyPBKDF2(),
	})
	if err != nil || solution == nil {
		t.Fatalf("SolveChallenge: %v (solution=%v)", err, solution)
	}
	b, err := json.Marshal(altcha.Payload{Challenge: challenge, Solution: *solution})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func newTestCaptchaService(t *testing.T, mutate func(*CaptchaConfig)) (*CaptchaService, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cfg := CaptchaConfig{
		Enabled:       true,
		Threshold:     3,
		ChallengeTTL:  5 * time.Minute,
		HMACSecret:    "test-hmac-secret",
		PoWCost:       1000,
		PoWDifficulty: 50,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	return NewCaptchaService(client, cfg), mr
}

func TestCaptchaService_VerifyValidSolution(t *testing.T) {
	svc, _ := newTestCaptchaService(t, nil)
	ctx := context.Background()

	challenge, err := svc.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if challenge.Signature == "" {
		t.Fatal("challenge must be signed")
	}
	if challenge.Parameters.ExpiresAt == 0 {
		t.Fatal("challenge must carry an expiry")
	}

	payload := solve(t, challenge)
	if !svc.Verify(ctx, payload) {
		t.Fatal("valid unconsumed solution must verify (AC-02)")
	}
	// Single-use: the same payload is a replay on the second attempt.
	if svc.Verify(ctx, payload) {
		t.Fatal("replayed solution must be rejected (AC-03)")
	}
}

func TestCaptchaService_VerifyRejectsBadPayloads(t *testing.T) {
	svc, _ := newTestCaptchaService(t, nil)
	ctx := context.Background()

	challenge, err := svc.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	valid := solve(t, challenge)

	// Forged: tamper with the signed parameters, breaking the HMAC.
	var payload altcha.Payload
	b, err := base64.StdEncoding.DecodeString(valid)
	if err != nil {
		t.Fatalf("decode valid payload: %v", err)
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("unmarshal valid payload: %v", err)
	}
	forged := payload
	forged.Challenge.Parameters.Cost++
	forgedB, err := json.Marshal(forged)
	if err != nil {
		t.Fatalf("marshal forged: %v", err)
	}

	// Mismatched pair: challenge A bundled with the solution of challenge B.
	// Each challenge has a unique salt/nonce, so B's key does not match A's
	// signed key prefix.
	other, err := svc.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate other: %v", err)
	}
	otherSolution, err := altcha.SolveChallenge(altcha.SolveChallengeOptions{
		Challenge: other,
		DeriveKey: altcha.DeriveKeyPBKDF2(),
	})
	if err != nil || otherSolution == nil {
		t.Fatalf("SolveChallenge other: %v (solution=%v)", err, otherSolution)
	}
	mismatched, err := json.Marshal(altcha.Payload{Challenge: challenge, Solution: *otherSolution})
	if err != nil {
		t.Fatalf("marshal mismatched: %v", err)
	}

	cases := []struct {
		name    string
		payload string
	}{
		{"absent", ""},
		{"not base64", "!!!not-base64!!!"},
		{"not json", base64.StdEncoding.EncodeToString([]byte("not-json"))},
		{"oversized", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", maxCaptchaPayloadSize+1)))},
		{"forged signature", base64.StdEncoding.EncodeToString(forgedB)},
		{"solution of another challenge", base64.StdEncoding.EncodeToString(mismatched)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if svc.Verify(ctx, tc.payload) {
				t.Fatalf("%s payload must be rejected (AC-03)", tc.name)
			}
		})
	}
}

func TestCaptchaService_VerifyRejectsExpiredChallenge(t *testing.T) {
	// The constructor normalizes non-positive TTLs to the default, so expire
	// by waiting out a short TTL instead.
	svc, _ := newTestCaptchaService(t, func(cfg *CaptchaConfig) {
		cfg.ChallengeTTL = 1 * time.Second
	})
	challenge, err := svc.Generate(context.Background())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	payload := solve(t, challenge)
	time.Sleep(1100 * time.Millisecond)
	if svc.Verify(context.Background(), payload) {
		t.Fatal("expired solution must be rejected (AC-03)")
	}
}

func TestCaptchaService_FailsClosedWithoutSecret(t *testing.T) {
	svc, _ := newTestCaptchaService(t, func(cfg *CaptchaConfig) {
		cfg.HMACSecret = ""
	})
	if _, err := svc.Generate(context.Background()); err == nil {
		t.Fatal("Generate must fail without an HMAC secret (boot validation backs this up)")
	}
	if svc.Verify(context.Background(), "anything") {
		t.Fatal("Verify must fail closed without an HMAC secret")
	}
}

func TestCaptchaService_RequiredForFailures(t *testing.T) {
	enabled, _ := newTestCaptchaService(t, nil)
	if enabled.RequiredForFailures(2) {
		t.Fatal("below threshold must not require captcha")
	}
	if !enabled.RequiredForFailures(3) {
		t.Fatal("at threshold must require captcha")
	}
	disabled, _ := newTestCaptchaService(t, func(cfg *CaptchaConfig) {
		cfg.Enabled = false
	})
	if disabled.RequiredForFailures(100) {
		t.Fatal("disabled captcha must never be required")
	}
}

func TestCaptchaService_ChallengesAreUnique(t *testing.T) {
	svc, _ := newTestCaptchaService(t, nil)
	c1, err := svc.Generate(context.Background())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	c2, err := svc.Generate(context.Background())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if c1.Parameters.Salt == c2.Parameters.Salt || c1.Parameters.Nonce == c2.Parameters.Nonce {
		t.Fatal("two challenges must not share salt/nonce")
	}
	if c1.Signature == c2.Signature {
		t.Fatal("two challenges must not share a signature")
	}
}
