package zalo

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

// TestProvider_RotationHook_FiresOnRefresh pins the chatbot push contract: a
// forced refresh (admin "Kiểm tra kết nối", the daily renew job) invokes the
// hook with exactly the access token that was persisted.
func TestProvider_RotationHook_FiresOnRefresh(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	hooked := make(chan string, 1)
	p.SetRotationHook(func(_ context.Context, accessToken string) {
		hooked <- accessToken
	})

	if err := p.RefreshNow(context.Background()); err != nil {
		t.Fatalf("RefreshNow err: %v", err)
	}

	select {
	case got := <-hooked:
		if got == "" {
			t.Fatal("hook received an empty access token")
		}
		if got == "live-access" {
			t.Error("hook received the OLD access token; want the rotated one")
		}
		if stored := creds.snapshot().AccessToken; stored != got {
			t.Errorf("hook token %q != persisted token %q", got, stored)
		}
		if _, oauth := mock.counts(); oauth != 1 {
			t.Errorf("oauth calls = %d, want 1", oauth)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("rotation hook was not invoked after a successful refresh")
	}
}

// TestProvider_RotationHook_FiresOnSendRetryRefresh covers the -124 lane: the
// reactive refresh that rescues a send must push the new token too, since the
// chatbot keeps sending with whatever it last received. The retry here still
// draws -124 (business error, Go error nil) — the rotation itself succeeded.
func TestProvider_RotationHook_FiresOnSendRetryRefresh(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	mock.mu.Lock()
	mock.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		mock.sendCalls++
		mock.mu.Unlock()
		_, _ = io.WriteString(w, `{"error":-124,"message":"bad token"}`)
	}
	mock.mu.Unlock()
	hooked := make(chan string, 1)
	p.SetRotationHook(func(_ context.Context, accessToken string) {
		hooked <- accessToken
	})

	res, err := p.Send(context.Background(), "0987654321", "617976", "track-hook", map[string]string{
		"otp_code": "123456",
	})
	if err != nil {
		t.Fatalf("Send err: %v (a -124 after a successful refresh is a business error)", err)
	}
	if res.ErrorCode != ErrBadAccessToken {
		t.Errorf("ErrorCode = %d, want %d", res.ErrorCode, ErrBadAccessToken)
	}

	select {
	case got := <-hooked:
		if stored := creds.snapshot().AccessToken; stored != got {
			t.Errorf("hook token %q != persisted token %q", got, stored)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("rotation hook was not invoked after the -124 refresh")
	}
}

// TestProvider_RotationHook_SkippedWhenNoRotation: a caller that waited for the
// lock and reused the winner's already-persisted token must NOT push again —
// the winning request owns the push for that rotation. The winner's token lands
// while the loser's request is being served (-124), i.e. after it read creds
// but before its retry acquires refresh authority.
func TestProvider_RotationHook_SkippedWhenNoRotation(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	hooked := make(chan string, 1)
	p.SetRotationHook(func(_ context.Context, accessToken string) {
		hooked <- accessToken
	})

	mock.mu.Lock()
	seeded := false
	mock.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		mock.sendCalls++
		first := !seeded
		seeded = true
		mock.mu.Unlock()
		if first {
			// Concurrent winner: another request persists a different access
			// token while this request is being rejected with -124.
			winner := creds.snapshot()
			winner.AccessToken = "winner-access"
			if err := creds.UpdateTokens(r.Context(), winner); err != nil {
				t.Errorf("seed winner token: %v", err)
			}
			_, _ = io.WriteString(w, `{"error":-124,"message":"bad token"}`)
			return
		}
		_, _ = io.WriteString(w, `{"error":0,"data":{"msg_id":"m-winner-retry"}}`)
	}
	mock.mu.Unlock()

	res, err := p.Send(context.Background(), "0987654321", "617976", "track-winner", map[string]string{
		"otp_code": "123456",
	})
	if err != nil {
		t.Fatalf("Send err: %v", err)
	}
	if res.ErrorCode != 0 {
		t.Errorf("ErrorCode = %d, want 0 (retry succeeds with the winner's token)", res.ErrorCode)
	}

	select {
	case got := <-hooked:
		t.Fatalf("hook fired on a non-rotation with %q — the winner owns this push", got)
	case <-time.After(200 * time.Millisecond):
	}
	if stored := creds.snapshot().AccessToken; stored != "winner-access" {
		t.Errorf("stored token = %q, want the winner's token untouched", stored)
	}
}

// TestProvider_RotationHook_PanicDoesNotFailRefresh: the hook runs in the
// send/renew path, so a broken pusher must never turn a persisted rotation
// into a caller-visible failure.
func TestProvider_RotationHook_PanicDoesNotFailRefresh(t *testing.T) {
	p, _, creds, _ := newProviderWithMock(t)
	p.SetRotationHook(func(_ context.Context, _ string) {
		panic("pusher exploded")
	})

	if err := p.RefreshNow(context.Background()); err != nil {
		t.Fatalf("RefreshNow err = %v, want nil (hook panic must be contained)", err)
	}
	if stored := creds.snapshot().AccessToken; stored == "live-access" {
		t.Error("rotation did not persist the new token")
	}
}

// TestProvider_RotationHook_NoHookStillWorks guards the nil-hook default so
// tools that never wire a pusher behave exactly as before.
func TestProvider_RotationHook_NoHookStillWorks(t *testing.T) {
	p, _, _, _ := newProviderWithMock(t)
	if err := p.RefreshNow(context.Background()); err != nil {
		t.Fatalf("RefreshNow err: %v", err)
	}
}
