package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// chatFixture is a minimal valid chat.completions body.
const chatFixture = `{
	"id": "chatcmpl-1",
	"model": "mimo-v2.6-flash",
	"created": 1770000000,
	"choices": [
		{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Chào bạn",
				"tool_calls": [
					{
						"id": "call_1",
						"type": "function",
						"function": {"name": "get_payroll", "arguments": "{\"month\":\"2026-10\"}"}
					}
				]
			},
			"finish_reason": "tool_calls"
		}
	],
	"usage": {"prompt_tokens": 12, "completion_tokens": 7, "total_tokens": 19}
}`

func newTestClient(t *testing.T, srv *httptest.Server, mutate func(*Config)) *HTTPClient {
	t.Helper()
	cfg := Config{
		BaseURL:        srv.URL + "/v1",
		APIKey:         "test-key",
		HTTPTimeout:    2 * time.Second,
		MaxRetries:     0,
		RetryBaseDelay: time.Millisecond,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestChatSendsOpenAICompatibleRequest(t *testing.T) {
	var gotPath, gotAuth, gotContentType string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(chatFixture))
	}))
	defer srv.Close()

	temperature := 0.2
	resp, err := newTestClient(t, srv, nil).Chat(context.Background(), ChatRequest{
		Model: "mimo-v2.6-pro", // per-call model must reach the wire verbatim
		Messages: []Message{
			{Role: RoleSystem, Content: []ContentPart{TextPart("Bạn là trợ lý lương.")}},
			{Role: RoleUser, Content: []ContentPart{
				TextPart("Đọc bảng tính này"),
				ImagePart("data:image/png;base64,iVBORw0KGgo="),
			}},
		},
		Tools: []Tool{{
			Type: "function",
			Function: &FunctionDef{
				Name:        "get_payroll",
				Description: "Tra cứu bảng lương",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"month":{"type":"string"}}}`),
			},
		}},
		ToolChoice:  "auto",
		MaxTokens:   512,
		Temperature: &temperature,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if gotPath != "/v1/chat/completions" {
		t.Errorf("path = %q, want /v1/chat/completions", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want Bearer test-key", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotBody["model"] != "mimo-v2.6-pro" {
		t.Errorf("model = %v, want mimo-v2.6-pro", gotBody["model"])
	}
	if gotBody["tool_choice"] != "auto" || gotBody["max_tokens"] != float64(512) {
		t.Errorf("tool_choice/max_tokens = %v/%v, want auto/512", gotBody["tool_choice"], gotBody["max_tokens"])
	}
	if _, ok := gotBody["stream"]; ok {
		t.Errorf("stream must be omitted/forced false on the wire, body: %v", gotBody)
	}

	// System message (single text part) marshals to a plain string; the
	// multimodal user message marshals to a parts array with the base64 image.
	messages, ok := gotBody["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %v, want 2 entries", gotBody["messages"])
	}
	systemMsg := messages[0].(map[string]any)
	if systemMsg["content"] != "Bạn là trợ lý lương." {
		t.Errorf("system content = %v, want plain string", systemMsg["content"])
	}
	userMsg := messages[1].(map[string]any)
	parts, ok := userMsg["content"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("user content = %v, want 2 parts", userMsg["content"])
	}
	image := parts[1].(map[string]any)
	imageURL := image["image_url"].(map[string]any)
	if image["type"] != "image_url" || !strings.HasPrefix(imageURL["url"].(string), "data:image/png;base64,") {
		t.Errorf("image part = %v, want base64 data URL", image)
	}

	// Response parsing: choices, tool calls, usage.
	if len(resp.Choices) != 1 {
		t.Fatalf("choices = %d, want 1", len(resp.Choices))
	}
	choice := resp.Choices[0]
	if choice.FinishReason != "tool_calls" || choice.Message.Content != "Chào bạn" {
		t.Errorf("choice = %+v", choice)
	}
	if len(choice.Message.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(choice.Message.ToolCalls))
	}
	call := choice.Message.ToolCalls[0]
	if call.ID != "call_1" || call.Function.Name != "get_payroll" || call.Function.Arguments != `{"month":"2026-10"}` {
		t.Errorf("tool call = %+v", call)
	}
	if resp.Usage.TotalTokens != 19 {
		t.Errorf("usage = %+v, want total 19", resp.Usage)
	}
}

func TestMessageMarshalShapes(t *testing.T) {
	single, err := json.Marshal(Message{Role: RoleUser, Content: []ContentPart{TextPart("xin chào")}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(single) != `{"role":"user","content":"xin chào"}` {
		t.Errorf("text-only message = %s, want plain-string content", single)
	}

	toolOnly, err := json.Marshal(Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Function: ToolCallFunction{Name: "f", Arguments: "{}"}}}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(toolOnly), `"content":null`) {
		t.Errorf("tool-call message = %s, want null content", toolOnly)
	}

	multi, err := json.Marshal(Message{Role: RoleUser, Content: []ContentPart{TextPart("x"), ImagePart("data:image/jpeg;base64,AAAA")}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(multi), `"image_url"`) || !strings.Contains(string(multi), `"type":"text"`) {
		t.Errorf("multimodal message = %s, want parts array", multi)
	}
}

func TestStreamChunkParsing(t *testing.T) {
	// One SSE `data: {...}` payload of a streaming response. Consumption of
	// the SSE stream is not implemented yet; this pins the parsing contract.
	payload := `{
		"id": "chatcmpl-2",
		"model": "mimo-v2.6-flash",
		"choices": [{
			"index": 0,
			"delta": {"content": "Xin", "tool_calls": [{"index": 0, "function": {"name": "f", "arguments": ""}}]},
			"finish_reason": null
		}]
	}`
	var chunk StreamChunk
	if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
		t.Fatalf("unmarshal chunk: %v", err)
	}
	if len(chunk.Choices) != 1 || chunk.Choices[0].Delta.Content != "Xin" {
		t.Fatalf("chunk = %+v", chunk)
	}
	if len(chunk.Choices[0].Delta.ToolCalls) != 1 || chunk.Choices[0].Delta.ToolCalls[0].Function.Name != "f" {
		t.Fatalf("chunk tool call delta = %+v", chunk.Choices[0].Delta.ToolCalls)
	}
	if chunk.Choices[0].FinishReason != nil {
		t.Fatalf("finish_reason = %v, want nil mid-stream", chunk.Choices[0].FinishReason)
	}
}

func TestChatErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantKind ErrorKind
		wantMsg  string // Vietnamese fragment in the surfaced message
	}{
		{"invalid key", 401, `{"error":{"message":"Invalid API key","type":"invalid_request_error"}}`, KindAuth, "Khóa API MiMo không hợp lệ"},
		{"forbidden key", 403, `{"error":{"message":"forbidden"}}`, KindAuth, "Khóa API MiMo không hợp lệ"},
		{"retired model", 404, `{"error":{"message":"model not found"}}`, KindModelNotFound, "Không tìm thấy mô hình"},
		{"bad payload", 400, `{"error":{"message":"messages required"}}`, KindInvalidRequest, "không hợp lệ"},
		{"rate limited", 429, `{"error":{"message":"slow down"}}`, KindRateLimited, "giới hạn tần suất"},
		{"provider outage", 503, `{"error":{"message":"upstream down"}}`, KindUnavailable, "gián đoạn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			_, err := newTestClient(t, srv, nil).Chat(context.Background(), ChatRequest{
				Model:    ModelFlash,
				Messages: []Message{{Role: RoleUser, Content: []ContentPart{TextPart("hi")}}},
			})
			if err == nil {
				t.Fatal("Chat = nil error, want failure")
			}
			if kind := KindOf(err); kind != tt.wantKind {
				t.Fatalf("kind = %v, want %v (err: %v)", kind, tt.wantKind, err)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("message = %q, want Vietnamese fragment %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestChatTimeoutMapsToTimeoutKind(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(chatFixture))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv, func(c *Config) {
		c.HTTPTimeout = 20 * time.Millisecond
	}).Chat(context.Background(), ChatRequest{
		Model:    ModelFlash,
		Messages: []Message{{Role: RoleUser, Content: []ContentPart{TextPart("hi")}}},
	})
	if err == nil {
		t.Fatal("Chat = nil error, want timeout")
	}
	if kind := KindOf(err); kind != KindTimeout {
		t.Fatalf("kind = %v, want %v (err: %v)", kind, KindTimeout, err)
	}
	if !strings.Contains(err.Error(), "hết thời gian chờ") {
		t.Fatalf("message = %q, want the Vietnamese timeout state", err.Error())
	}
}

func TestChatRetriesTransientFailuresOnly(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(chatFixture))
	}))
	defer srv.Close()

	resp, err := newTestClient(t, srv, func(c *Config) {
		c.MaxRetries = 2
	}).Chat(context.Background(), ChatRequest{
		Model:    ModelFlash,
		Messages: []Message{{Role: RoleUser, Content: []ContentPart{TextPart("hi")}}},
	})
	if err != nil {
		t.Fatalf("Chat after retries: %v", err)
	}
	if resp.Choices[0].Message.Content != "Chào bạn" {
		t.Fatalf("resp = %+v", resp)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("attempts = %d, want 3 (two transient retries)", got)
	}

	// Auth failures must NOT be retried — the key will not fix itself.
	attempts.Store(0)
	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer authSrv.Close()

	_, err = newTestClient(t, authSrv, func(c *Config) { c.MaxRetries = 2 }).Chat(context.Background(), ChatRequest{
		Model:    ModelFlash,
		Messages: []Message{{Role: RoleUser, Content: []ContentPart{TextPart("hi")}}},
	})
	if err == nil || KindOf(err) != KindAuth {
		t.Fatalf("err = %v, want %v", err, KindAuth)
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("auth attempts = %d, want 1 (no retry on auth failure)", got)
	}
}

func TestChatValidatesRequestShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("no request should be sent for an invalid payload")
	}))
	defer srv.Close()
	client := newTestClient(t, srv, nil)

	if _, err := client.Chat(context.Background(), ChatRequest{Model: "", Messages: []Message{{Role: RoleUser}}}); err == nil {
		t.Error("Chat without model = nil error")
	}
	if _, err := client.Chat(context.Background(), ChatRequest{Model: ModelFlash}); err == nil {
		t.Error("Chat without messages = nil error")
	}
}

func TestTestConnection(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"mimo-v2.6-flash"},{"id":"mimo-v2.6-pro"}]}`))
	}))
	defer srv.Close()

	result, err := newTestClient(t, srv, nil).TestConnection(context.Background())
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	if gotPath != "/v1/models" {
		t.Errorf("path = %q, want /v1/models", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want Bearer test-key", gotAuth)
	}
	if result.ModelCount != 2 || len(result.Models) != 2 {
		t.Fatalf("result = %+v, want 2 models", result)
	}
	if !result.DefaultModelPresent {
		t.Error("DefaultModelPresent = false, want true when the default model is offered")
	}
	if result.Latency <= 0 || result.LatencyMS < 0 {
		t.Errorf("latency = %v/%d, want a measured round trip", result.Latency, result.LatencyMS)
	}
}

func TestTestConnectionInvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv, nil).TestConnection(context.Background())
	if err == nil || KindOf(err) != KindAuth {
		t.Fatalf("err = %v, want %v", err, KindAuth)
	}
}

func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	if _, err := NewClient(Config{BaseURL: "not a url"}); err == nil {
		t.Fatal("NewClient with a malformed base URL = nil error")
	}
	if _, err := NewClient(Config{BaseURL: "ftp://example.com"}); err == nil {
		t.Fatal("NewClient with a non-HTTP scheme = nil error")
	}
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("NewClient defaults: %v", err)
	}
	if client.cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("default BaseURL = %q, want %q", client.cfg.BaseURL, DefaultBaseURL)
	}
}
