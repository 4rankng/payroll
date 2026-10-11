package llm

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	llmport "api-server/internal/infra/llm"
)

// fakeSettings is the SettingsSource seam with plain fields.
type fakeSettings struct {
	key          string
	baseURL      string
	defaultModel string
	fallback     string
}

func (f *fakeSettings) GetMimoAPIKey(context.Context) string                    { return f.key }
func (f *fakeSettings) GetMimoBaseURL(context.Context) string                   { return f.baseURL }
func (f *fakeSettings) GetMimoDefaultModel(context.Context) string              { return f.defaultModel }
func (f *fakeSettings) GetMimoFallbackModel(context.Context) string             { return f.fallback }
func (f *fakeSettings) GetMimoModelForRoute(_ context.Context, _ string) string { return "" }

// fakeClient records calls and plays back scripted outcomes. Call recording is
// mutex-guarded — the busy-path test drives concurrent calls.
type fakeClient struct {
	mu       sync.Mutex
	chatFn   func(ctx context.Context, req llmport.ChatRequest) (*llmport.ChatResponse, error)
	testFn   func() (*llmport.TestConnectionResult, error)
	chatReqs []llmport.ChatRequest
}

func (f *fakeClient) Chat(ctx context.Context, req llmport.ChatRequest) (*llmport.ChatResponse, error) {
	f.mu.Lock()
	f.chatReqs = append(f.chatReqs, req)
	f.mu.Unlock()
	if f.chatFn != nil {
		return f.chatFn(ctx, req)
	}
	return &llmport.ChatResponse{Choices: []llmport.Choice{{Message: llmport.ResponseMessage{Content: "ok"}}}}, nil
}

func (f *fakeClient) TestConnection(context.Context) (*llmport.TestConnectionResult, error) {
	if f.testFn != nil {
		return f.testFn()
	}
	return &llmport.TestConnectionResult{LatencyMS: 12, ModelCount: 2}, nil
}

// newTestService wires a Service around the fakes and captures the client
// config the service derives from settings. The capture is mutex-guarded: the
// busy-path test drives concurrent calls.
func newTestService(settings *fakeSettings, client *fakeClient, guard *llmport.Guard) (*Service, *llmport.Config) {
	var mu sync.Mutex
	var captured llmport.Config
	svc := NewService(settings, guard, func(cfg llmport.Config) (llmport.Client, error) {
		mu.Lock()
		captured = cfg
		mu.Unlock()
		return client, nil
	})
	return svc, &captured
}

func TestTestConnectionMissingKey(t *testing.T) {
	svc, _ := newTestService(&fakeSettings{}, &fakeClient{}, nil)

	_, err := svc.TestConnection(context.Background())
	if err == nil {
		t.Fatal("TestConnection without a key = nil error")
	}
	if kind := llmport.KindOf(err); kind != llmport.KindMissingKey {
		t.Fatalf("kind = %v, want %v", kind, llmport.KindMissingKey)
	}
	if err.Error() != llmport.MessageFor(llmport.KindMissingKey) {
		t.Fatalf("message = %q, want the designed missing-key state", err.Error())
	}
}

func TestTestConnectionBuildsClientFromSettings(t *testing.T) {
	client := &fakeClient{testFn: func() (*llmport.TestConnectionResult, error) {
		return &llmport.TestConnectionResult{LatencyMS: 234, ModelCount: 2, DefaultModelPresent: true}, nil
	}}
	svc, cfg := newTestService(&fakeSettings{key: "sk-secret", baseURL: "https://example.com/v1"}, client, nil)

	result, err := svc.TestConnection(context.Background())
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	if result.LatencyMS != 234 || result.ModelCount != 2 || !result.DefaultModelPresent {
		t.Fatalf("result = %+v", result)
	}
	if cfg.APIKey != "sk-secret" || cfg.BaseURL != "https://example.com/v1" {
		t.Fatalf("client cfg = %+v, want the settings key/base URL", cfg)
	}
}

func TestChatResolvesModelFromSettingsDefault(t *testing.T) {
	client := &fakeClient{}
	svc, _ := newTestService(&fakeSettings{
		key: "sk", defaultModel: llmport.ModelPro, fallback: llmport.ModelFlash,
	}, client, nil)

	if _, err := svc.Chat(context.Background(), 7, llmport.ChatRequest{}); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if len(client.chatReqs) != 1 || client.chatReqs[0].Model != llmport.ModelPro {
		t.Fatalf("chat requests = %+v, want the settings default model", client.chatReqs)
	}
}

func TestChatRejectsRetiredModel(t *testing.T) {
	client := &fakeClient{}
	svc, _ := newTestService(&fakeSettings{key: "sk"}, client, nil)

	_, err := svc.Chat(context.Background(), 7, llmport.ChatRequest{Model: "mimo-v2.5-flash"})
	if err == nil {
		t.Fatal("Chat with a retired model = nil error")
	}
	if kind := llmport.KindOf(err); kind != llmport.KindModelNotFound {
		t.Fatalf("kind = %v, want %v", kind, llmport.KindModelNotFound)
	}
	if len(client.chatReqs) != 0 {
		t.Fatal("no request may reach the provider for a retired model")
	}
}

func TestChatFallsBackToFallbackModelOnModelNotFound(t *testing.T) {
	client := &fakeClient{chatFn: func(_ context.Context, req llmport.ChatRequest) (*llmport.ChatResponse, error) {
		if req.Model == llmport.ModelPro {
			return nil, &llmport.APIError{Kind: llmport.KindModelNotFound, Message: "model gone"}
		}
		return &llmport.ChatResponse{Choices: []llmport.Choice{{Message: llmport.ResponseMessage{Content: "fallback ok"}}}}, nil
	}}
	svc, _ := newTestService(&fakeSettings{
		key: "sk", defaultModel: llmport.ModelPro, fallback: llmport.ModelFlash,
	}, client, nil)

	resp, err := svc.Chat(context.Background(), 7, llmport.ChatRequest{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Choices[0].Message.Content != "fallback ok" {
		t.Fatalf("resp = %+v", resp)
	}
	if len(client.chatReqs) != 2 {
		t.Fatalf("calls = %d, want 2 (primary + fallback)", len(client.chatReqs))
	}
	if client.chatReqs[1].Model != llmport.ModelFlash {
		t.Fatalf("fallback model = %q, want %q", client.chatReqs[1].Model, llmport.ModelFlash)
	}
}

func TestChatRateLimitedPerUser(t *testing.T) {
	client := &fakeClient{}
	svc, _ := newTestService(&fakeSettings{key: "sk"}, client, llmport.NewGuard(
		llmport.NewSemaphore(4),
		llmport.NewMemoryLimiter(time.Minute, 1),
	))

	if _, err := svc.Chat(context.Background(), 42, llmport.ChatRequest{}); err != nil {
		t.Fatalf("first Chat: %v", err)
	}
	_, err := svc.Chat(context.Background(), 42, llmport.ChatRequest{})
	if err == nil {
		t.Fatal("second Chat over the per-user budget = nil error")
	}
	if kind := llmport.KindOf(err); kind != llmport.KindRateLimited {
		t.Fatalf("kind = %v, want %v", kind, llmport.KindRateLimited)
	}
	// Another user's budget is untouched.
	if _, err := svc.Chat(context.Background(), 43, llmport.ChatRequest{}); err != nil {
		t.Fatalf("other user Chat: %v", err)
	}
	if len(client.chatReqs) != 2 {
		t.Fatalf("provider calls = %d, want 2 (denied calls never reach the provider)", len(client.chatReqs))
	}
}

// errorLimiter simulates a broken guard (e.g. Redis down).
type errorLimiter struct{}

func (errorLimiter) Allow(context.Context, string) (bool, error) {
	return false, errors.New("redis: connection refused")
}

func TestChatFailsOpenWhenLimiterMalfunctions(t *testing.T) {
	client := &fakeClient{}
	svc, _ := newTestService(&fakeSettings{key: "sk"}, client, llmport.NewGuard(
		llmport.NewSemaphore(4), errorLimiter{},
	))

	if _, err := svc.Chat(context.Background(), 1, llmport.ChatRequest{}); err != nil {
		t.Fatalf("Chat with a broken limiter must fail open, got %v", err)
	}
	if len(client.chatReqs) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(client.chatReqs))
	}
}

func TestChatBusyWhenGuardSaturated(t *testing.T) {
	client := &fakeClient{chatFn: func(context.Context, llmport.ChatRequest) (*llmport.ChatResponse, error) {
		// Hold the only slot until the second caller times out.
		time.Sleep(80 * time.Millisecond)
		return &llmport.ChatResponse{Choices: []llmport.Choice{{}}}, nil
	}}
	svc, _ := newTestService(&fakeSettings{key: "sk"}, client, llmport.NewGuard(
		llmport.NewSemaphore(1), nil,
	))

	oldWait := GuardWait
	GuardWait = 20 * time.Millisecond
	defer func() { GuardWait = oldWait }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = svc.Chat(context.Background(), 1, llmport.ChatRequest{})
	}()
	time.Sleep(5 * time.Millisecond)

	_, err := svc.Chat(context.Background(), 2, llmport.ChatRequest{})
	if err == nil {
		t.Fatal("Chat on a saturated guard = nil error")
	}
	if kind := llmport.KindOf(err); kind != llmport.KindBusy {
		t.Fatalf("kind = %v, want %v (err: %v)", kind, llmport.KindBusy, err)
	}
	<-done
}

func TestChatTimeoutFromCallerContextPropagates(t *testing.T) {
	client := &fakeClient{chatFn: func(ctx context.Context, _ llmport.ChatRequest) (*llmport.ChatResponse, error) {
		select {
		case <-ctx.Done():
			return nil, llmport.ContextError(ctx)
		case <-time.After(50 * time.Millisecond):
			return &llmport.ChatResponse{Choices: []llmport.Choice{{}}}, nil
		}
	}}
	svc, _ := newTestService(&fakeSettings{key: "sk"}, client, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := svc.Chat(ctx, 1, llmport.ChatRequest{})
	if err == nil {
		t.Fatal("Chat with an expired caller context = nil error")
	}
	if kind := llmport.KindOf(err); kind != llmport.KindTimeout {
		t.Fatalf("kind = %v, want %v (err: %v)", kind, llmport.KindTimeout, err)
	}
}
