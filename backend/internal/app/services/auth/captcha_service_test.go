package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	altcha "github.com/altcha-org/altcha-lib-go/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCaptchaService(t *testing.T, ttl time.Duration) (*CaptchaService, *miniredis.Miniredis) {
	t.Helper()
	return newTestCaptchaServiceWithSecret(t, ttl, "test-captcha-hmac-secret")
}

func newTestCaptchaServiceWithSecret(t *testing.T, ttl time.Duration, secret string) (*CaptchaService, *miniredis.Miniredis) {
	t.Helper()
	redisServer := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewCaptchaService(rdb, CaptchaConfig{
		Enabled:      true,
		Threshold:    3,
		ChallengeTTL: ttl,
		HMACSecret:   secret,
		Cost:         1, // keep the test's own solve fast
	}), redisServer
}

// solve solves the challenge the same way the browser widget does, then
// returns the base64 payload a login request would carry.
func solve(t *testing.T, svc *CaptchaService) string {
	t.Helper()
	challenge, err := svc.Generate(context.Background())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	solution, err := altcha.SolveChallenge(altcha.SolveChallengeOptions{
		Challenge: challenge,
		DeriveKey: altcha.DeriveKeySHA(),
	})
	if err != nil || solution == nil {
		t.Fatalf("SolveChallenge: solution=%v err=%v", solution, err)
	}
	raw, err := json.Marshal(map[string]any{"challenge": challenge, "solution": *solution})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func TestCaptchaServiceVerifyHappyPathThenReplayRejected(t *testing.T) {
	svc, _ := newTestCaptchaService(t, 5*time.Minute)
	ctx := context.Background()

	payload := solve(t, svc)
	if !svc.Verify(ctx, payload) {
		t.Fatal("expected a freshly solved payload to verify")
	}
	if svc.Verify(ctx, payload) {
		t.Fatal("expected the same payload to be rejected as a replay")
	}
}

func TestCaptchaServiceVerifyRejectsBadInput(t *testing.T) {
	svc, _ := newTestCaptchaService(t, 5*time.Minute)
	ctx := context.Background()

	cases := map[string]string{
		"empty":    "",
		"garbage":  "not-base64-$$$",
		"not json": base64.StdEncoding.EncodeToString([]byte("hello")),
		"truncated payload": base64.StdEncoding.EncodeToString(
			[]byte(`{"challenge":{"parameters":{"algorithm":"SHA-256"}}}`),
		),
	}
	// A payload with a validly signed challenge but a WRONG proof-of-work
	// counter must be rejected: the signature covers the challenge parameters,
	// not the solver's work — the hash check itself is what rejects it.
	challenge, err := altcha.CreateChallenge(altcha.CreateChallengeOptions{
		Algorithm:           string(altcha.SHA256),
		DeriveKey:           altcha.DeriveKeySHA(),
		HMACSignatureSecret: "test-captcha-hmac-secret",
		Cost:                1,
	})
	if err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}
	solution, err := altcha.SolveChallenge(altcha.SolveChallengeOptions{Challenge: challenge, DeriveKey: altcha.DeriveKeySHA()})
	if err != nil || solution == nil {
		t.Fatalf("SolveChallenge: solution=%v err=%v", solution, err)
	}
	solution.Counter += 1 // valid signature, invalid work
	raw, err := json.Marshal(map[string]any{"challenge": challenge, "solution": *solution})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	cases["tampered counter"] = base64.StdEncoding.EncodeToString(raw)

	for name, payload := range cases {
		if svc.Verify(ctx, payload) {
			t.Errorf("%s: expected rejection", name)
		}
	}

	// A payload signed for a different secret must fail (forged challenge).
	other, _ := newTestCaptchaServiceWithSecret(t, 5*time.Minute, "a-different-hmac-secret")
	payload := solve(t, other)
	if svc.Verify(ctx, payload) {
		t.Error("expected a payload signed with another secret to be rejected")
	}
}

func TestCaptchaServiceVerifyRejectsExpiredChallenge(t *testing.T) {
	svc, _ := newTestCaptchaService(t, 5*time.Minute)
	ctx := context.Background()

	// Craft a challenge whose signature is valid but whose expiry is in the
	// past — the server must reject it even though it solves correctly.
	past := time.Now().UTC().Add(-time.Minute)
	challenge, err := altcha.CreateChallenge(altcha.CreateChallengeOptions{
		Algorithm:           string(altcha.SHA256),
		DeriveKey:           altcha.DeriveKeySHA(),
		HMACSignatureSecret: "test-captcha-hmac-secret",
		Cost:                1,
		ExpiresAt:           &past,
	})
	if err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}
	solution, err := altcha.SolveChallenge(altcha.SolveChallengeOptions{
		Challenge: challenge,
		DeriveKey: altcha.DeriveKeySHA(),
	})
	if err != nil || solution == nil {
		t.Fatalf("SolveChallenge: solution=%v err=%v", solution, err)
	}
	raw, err := json.Marshal(map[string]any{"challenge": challenge, "solution": *solution})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if svc.Verify(ctx, base64.StdEncoding.EncodeToString(raw)) {
		t.Fatal("expected an expired challenge to be rejected")
	}
}

func TestCaptchaServiceDisabledNeverRequired(t *testing.T) {
	redisServer := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	svc := NewCaptchaService(rdb, CaptchaConfig{Enabled: false, Threshold: 3, HMACSecret: "s"})

	if svc.RequiredForFailures(100) {
		t.Fatal("a disabled service must never require a captcha")
	}
}
