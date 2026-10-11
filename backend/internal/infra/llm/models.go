package llm

import "strings"

// Allowed MiMo model IDs. The vendor renamed its model family within 10 days of
// launch, so the allowlist is deliberately small and the effective models are
// runtime settings (mimo.default_model / mimo.fallback_model) validated against
// this list — a rename costs a settings edit, not a redeploy. Deprecated
// families (mimo-v2.5*) are rejected explicitly so an old config cannot route
// production traffic to a retired model.
const (
	// ModelFlash is the default model: fast drafting, chat, triage.
	ModelFlash = "mimo-v2.6-flash"
	// ModelPro is the deliverable model: long documents, high-stakes drafts.
	ModelPro = "mimo-v2.6-pro"

	// DefaultModel is the model used when no runtime override is configured.
	DefaultModel = ModelFlash
	// DefaultFallbackModel is used when the primary model is not found (the
	// vendor renamed it out from under us).
	DefaultFallbackModel = ModelPro
)

// AllowedModels is the complete set of model IDs accepted by config validation
// and call-time routing.
var AllowedModels = []string{ModelFlash, ModelPro}

// deprecatedModelPrefixes are model families that must never be configured or
// called. Matching is by prefix so the whole retired family is covered.
var deprecatedModelPrefixes = []string{"mimo-v2.5"}

// IsAllowedModel reports whether model is in the current allowlist.
func IsAllowedModel(model string) bool {
	for _, allowed := range AllowedModels {
		if model == allowed {
			return true
		}
	}
	return false
}

// ValidateModel rejects retired and unknown model IDs with a clear Vietnamese
// error. Empty is invalid here — ResolveModel applies the default instead.
func ValidateModel(model string) error {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return &APIError{Kind: KindInvalidRequest, Message: "Chưa chọn mô hình MiMo."}
	}
	for _, prefix := range deprecatedModelPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return &APIError{Kind: KindModelNotFound, Message: "Mô hình " + trimmed + " đã ngừng hỗ trợ — dùng " + ModelFlash + " hoặc " + ModelPro + "."}
		}
	}
	if !IsAllowedModel(trimmed) {
		return &APIError{Kind: KindModelNotFound, Message: "Mô hình " + trimmed + " không nằm trong danh sách cho phép (" + ModelFlash + ", " + ModelPro + ")."}
	}
	return nil
}

// ResolveModel resolves the model for one call: an explicit request wins, an
// empty request falls back to defaultValue, and an empty defaultValue falls
// back to DefaultModel. The result is validated, so a stale runtime override
// can never route traffic to a retired family.
func ResolveModel(requested, defaultValue string) (string, error) {
	model := strings.TrimSpace(requested)
	if model == "" {
		model = strings.TrimSpace(defaultValue)
	}
	if model == "" {
		model = DefaultModel
	}
	if err := ValidateModel(model); err != nil {
		return "", err
	}
	return model, nil
}
