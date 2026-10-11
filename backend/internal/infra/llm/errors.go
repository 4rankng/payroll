package llm

import (
	"errors"
	"strings"
)

// ErrorKind classifies provider failures so the transport/UI layers can render
// designed error states instead of raw error strings. The Vietnamese messages
// below are the user-facing contract; keep them stable.
type ErrorKind string

const (
	// KindMissingKey — no API key is configured in settings yet.
	KindMissingKey ErrorKind = "missing_key"
	// KindAuth — the provider rejected the API key (401/403).
	KindAuth ErrorKind = "invalid_key"
	// KindInvalidRequest — the provider rejected the payload (400/422).
	KindInvalidRequest ErrorKind = "invalid_request"
	// KindModelNotFound — the model ID no longer exists (vendor renamed its
	// model family); the app service retries with the fallback model.
	KindModelNotFound ErrorKind = "model_not_found"
	// KindRateLimited — provider-side rate limit (429) or our per-user guard.
	KindRateLimited ErrorKind = "rate_limited"
	// KindTimeout — the provider did not answer within the bounded timeout.
	KindTimeout ErrorKind = "timeout"
	// KindUnavailable — provider outage (5xx) or transport failure.
	KindUnavailable ErrorKind = "unavailable"
	// KindBusy — our concurrency guard rejected the call (shared-key
	// protection: one runaway loop must not starve the other consumers).
	KindBusy ErrorKind = "busy"
	// KindInternal — anything else.
	KindInternal ErrorKind = "error"
)

// APIError is a classified LLM failure. Message is always Vietnamese and safe
// to surface in the UI; providerStatus carries the HTTP status for logs.
type APIError struct {
	Kind           ErrorKind
	Message        string
	ProviderStatus int
	// RequestID is the provider's request id when the response carried one —
	// for support, never for display.
	RequestID string
}

// Error implements the error interface. It returns the Vietnamese
// user-facing message; Kind is inspected via KindOf for programmatic handling.
func (e *APIError) Error() string {
	return e.Message
}

// KindOf extracts the ErrorKind from an LLM error, or KindInternal for
// unclassified errors.
func KindOf(err error) ErrorKind {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Kind
	}
	return KindInternal
}

// MessageFor returns the Vietnamese user-facing message for a kind, with a
// short context suffix when one applies.
func MessageFor(kind ErrorKind) string {
	switch kind {
	case KindMissingKey:
		return "Chưa cấu hình Khóa bí mật MiMo trong Cài đặt — nhập khóa trước khi sử dụng trợ lý AI."
	case KindAuth:
		return "Khóa API MiMo không hợp lệ hoặc đã bị thu hồi — kiểm tra lại Khóa bí mật trong Cài đặt."
	case KindInvalidRequest:
		return "Yêu cầu tới MiMo không hợp lệ — vui lòng thử lại hoặc báo quản trị kiểm tra cấu hình."
	case KindModelNotFound:
		return "Không tìm thấy mô hình MiMo đã cấu hình — nhà cung cấp có thể đã đổi tên mô hình; kiểm tra cấu hình mô hình trong Cài đặt."
	case KindRateLimited:
		return "Đã vượt giới hạn tần suất yêu cầu tới MiMo — thử lại sau ít phút."
	case KindTimeout:
		return "MiMo không phản hồi kịp (hết thời gian chờ) — thử lại sau ít phút."
	case KindUnavailable:
		return "Dịch vụ MiMo đang gián đoạn — thử lại sau ít phút. Các nghiệp vụ khác không bị ảnh hưởng."
	case KindBusy:
		return "Hệ thống đang xử lý nhiều yêu cầu AI cùng lúc — thử lại sau ít phút."
	default:
		return "Không thể kết nối MiMo — thử lại sau ít phút."
	}
}

// newAPIError builds an APIError with the canonical Vietnamese message for the
// kind. The provider's own error text is appended (bounded) so support can see
// what the vendor said without leaking it as the primary message.
func newAPIError(kind ErrorKind, providerStatus int, providerMessage, requestID string) *APIError {
	msg := MessageFor(kind)
	trimmed := strings.TrimSpace(providerMessage)
	if trimmed != "" && kind != KindMissingKey {
		const maxProviderMsg = 200
		if len(trimmed) > maxProviderMsg {
			trimmed = trimmed[:maxProviderMsg] + "…"
		}
		msg += " (" + trimmed + ")"
	}
	return &APIError{
		Kind:           kind,
		Message:        msg,
		ProviderStatus: providerStatus,
		RequestID:      requestID,
	}
}

// errorKindForStatus maps an HTTP status from the provider to an ErrorKind.
// 404 is treated as model-not-found because OpenAI-compatible gateways answer
// unknown model IDs with 404; the app service retries with the fallback model.
func errorKindForStatus(status int) ErrorKind {
	switch {
	case status == 401 || status == 403:
		return KindAuth
	case status == 404:
		return KindModelNotFound
	case status == 429:
		return KindRateLimited
	case status == 400 || status == 422:
		return KindInvalidRequest
	case status >= 500:
		return KindUnavailable
	default:
		return KindInternal
	}
}
