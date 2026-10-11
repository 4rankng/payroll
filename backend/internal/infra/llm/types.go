// Package llm is the LLM provider seam. It holds the narrow port application
// services depend on (Client) next to the concrete OpenAI-compatible HTTP
// client for Xiaomi MiMo (https://token-plan-sgp.xiaomimimo.com/v1 chat.completions).
// Keeping both here mirrors internal/infra/zalo: the port is small, the
// protocol details (auth header, error bodies, retries) stay inside the
// package, and swapping providers is a bootstrap wiring change — the app
// services only ever see the interface.
//
// The request/response structs below cover function calling, image input
// (base64 data URLs), and streaming-capable chunk parsing. SSE consumption is
// not implemented yet (the assistant MVP is non-streaming) but the chunk
// structs parse a single `data: {...}` payload so streaming lands without a
// contract change.
package llm

import (
	"context"
	"encoding/json"
	"time"
)

// Client is the port application services depend on. Implementations must be
// safe for concurrent use.
type Client interface {
	// Chat runs one chat.completions turn. The model is per-call; the
	// implementation validates nothing about it beyond transport concerns
	// (model routing is owned by the app service).
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	// TestConnection validates the configured credentials against the provider
	// (GET /models for OpenAI-compatible endpoints) and reports latency.
	TestConnection(ctx context.Context) (*TestConnectionResult, error)
}

// Message roles, mirroring the OpenAI chat.completions vocabulary.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Content part types for multimodal (text + image) message content.
const (
	ContentPartText     = "text"
	ContentPartImageURL = "image_url"
)

// ImageURL is one image content part. URL is a data URL
// ("data:image/png;base64,...") for inline base64 input, or a remote URL.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// ContentPart is one block of message content: text or an image.
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// TextPart builds a text content part.
func TextPart(text string) ContentPart {
	return ContentPart{Type: ContentPartText, Text: text}
}

// ImagePart builds an image content part from a base64 data URL.
func ImagePart(dataURL string) ContentPart {
	return ContentPart{Type: ContentPartImageURL, ImageURL: &ImageURL{URL: dataURL}}
}

// ToolCallFunction is the function name + JSON-encoded arguments of one tool
// call. Arguments arrive as a string on responses and are echoed back verbatim
// on follow-up assistant turns.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall is one function call requested by the model.
type ToolCall struct {
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function"`
}

// FunctionDef declares one callable function (JSON Schema parameters).
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// Tool declares one tool the model may call.
type Tool struct {
	Type     string       `json:"type,omitempty"`
	Function *FunctionDef `json:"function,omitempty"`
}

// Message is a request-side chat message. Content marshals to the wire form
// the OpenAI-compatible API expects: a plain string for simple text, an array
// of parts for multimodal content.
type Message struct {
	Role       string        `json:"role"`
	Content    []ContentPart `json:"-"`
	ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

// MarshalJSON emits the wire form: text-only content becomes a plain string
// (the minimal request shape), multimodal content becomes a part array, and a
// tool-call-only assistant message carries null content.
func (m Message) MarshalJSON() ([]byte, error) {
	wire := struct {
		Role       string     `json:"role"`
		Content    any        `json:"content"`
		ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
		ToolCallID string     `json:"tool_call_id,omitempty"`
	}{Role: m.Role, ToolCalls: m.ToolCalls, ToolCallID: m.ToolCallID}

	switch len(m.Content) {
	case 0:
		wire.Content = nil
	case 1:
		if m.Content[0].Type == ContentPartText {
			wire.Content = m.Content[0].Text
		} else {
			wire.Content = m.Content
		}
	default:
		wire.Content = m.Content
	}
	return json.Marshal(wire)
}

// ChatRequest is one chat.completions call. Model is required and resolved per
// call (the model family moved once within 10 days of launch — routing is a
// runtime setting, not a constant baked into callers).
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"` // "auto" | "none" | "required"
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// ResponseMessage is one assistant message returned by the API. Content is a
// plain string on the wire; tool calls carry JSON-encoded arguments.
type ResponseMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Choice is one completion alternative.
type Choice struct {
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

// Usage reports token accounting for the call.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse is a full (non-streaming) chat.completions response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Created int64    `json:"created"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// ToolCallDelta is a partial tool call in a streaming chunk; Index stitches
// fragments across chunks.
type ToolCallDelta struct {
	Index    int                   `json:"index"`
	ID       string                `json:"id,omitempty"`
	Type     string                `json:"type,omitempty"`
	Function ToolCallFunctionDelta `json:"function,omitempty"`
}

// ToolCallFunctionDelta is a partial function name/arguments fragment.
type ToolCallFunctionDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// StreamDelta is the incremental payload of one streaming choice.
type StreamDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []ToolCallDelta `json:"tool_calls,omitempty"`
}

// StreamChoice is one choice fragment of a streaming chunk.
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// StreamChunk parses a single SSE `data: {...}` payload of a streaming
// chat.completions response. Consumption of the SSE stream itself is not
// implemented yet; this struct exists so the parsing contract is stable when
// streaming lands.
type StreamChunk struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Created int64          `json:"created"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// TestConnectionResult is the outcome of validating stored credentials against
// the provider. Models lists the model IDs the provider reports.
type TestConnectionResult struct {
	Latency    time.Duration `json:"-"`
	LatencyMS  int64         `json:"latency_ms"`
	Models     []string      `json:"models,omitempty"`
	ModelCount int           `json:"model_count"`
	// DefaultModelPresent reports whether the configured default model is
	// among the provider's models — the cheapest signal that the model family
	// was renamed under us.
	DefaultModelPresent bool `json:"default_model_present"`
}

// Config holds provider-level knobs. Credentials live here too: the API key is
// read from settings per call (admins rotate it at runtime), so clients are
// cheap values built around a shared http.Transport.
type Config struct {
	// BaseURL is the OpenAI-compatible root including the version segment,
	// e.g. https://token-plan-sgp.xiaomimimo.com/v1. Empty means DefaultBaseURL.
	BaseURL string
	// APIKey is sent as `Authorization: Bearer <key>`.
	APIKey string
	// HTTPTimeout bounds each HTTP attempt. Empty means DefaultHTTPTimeout.
	HTTPTimeout time.Duration
	// MaxRetries is the number of RETRIES after the first attempt, and only
	// for transient failures (transport errors, 429, 5xx). Empty means
	// DefaultMaxRetries.
	MaxRetries int
	// RetryBaseDelay is the exponential-backoff base between retries. Empty
	// means DefaultRetryBaseDelay.
	RetryBaseDelay time.Duration
}
