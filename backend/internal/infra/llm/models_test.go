package llm

import (
	"strings"
	"testing"
)

func TestValidateModelAcceptsAllowlist(t *testing.T) {
	for _, model := range AllowedModels {
		if err := ValidateModel(model); err != nil {
			t.Fatalf("ValidateModel(%q) = %v, want nil", model, err)
		}
	}
	if IsAllowedModel(ModelFlash) != true || IsAllowedModel(ModelPro) != true {
		t.Fatal("allowlist models must report as allowed")
	}
}

func TestValidateModelRejectsDeprecatedV25Family(t *testing.T) {
	deprecated := []string{
		"mimo-v2.5",
		"mimo-v2.5-flash",
		"mimo-v2.5-pro",
		"mimo-v2.5-flash-20260101",
	}
	for _, model := range deprecated {
		err := ValidateModel(model)
		if err == nil {
			t.Fatalf("ValidateModel(%q) = nil, want rejection", model)
		}
		if kind := KindOf(err); kind != KindModelNotFound {
			t.Fatalf("ValidateModel(%q) kind = %v, want %v", model, kind, KindModelNotFound)
		}
		// The rejection must be a clear Vietnamese error naming the family and
		// the supported replacements.
		msg := err.Error()
		if !strings.Contains(msg, "mimo-v2.5") || !strings.Contains(msg, "ngừng hỗ trợ") {
			t.Fatalf("ValidateModel(%q) message = %q, want a clear Vietnamese deprecation error", model, msg)
		}
		if !strings.Contains(msg, ModelFlash) || !strings.Contains(msg, ModelPro) {
			t.Fatalf("ValidateModel(%q) message = %q, want the supported models listed", model, msg)
		}
	}
	if IsAllowedModel("mimo-v2.5-flash") {
		t.Fatal("a retired model must never be in the allowlist")
	}
}

func TestValidateModelRejectsUnknownAndEmpty(t *testing.T) {
	for _, model := range []string{"gpt-4o", "mimo-v3-ultra", "mimo-v2.6-flash-evil"} {
		if err := ValidateModel(model); err == nil {
			t.Fatalf("ValidateModel(%q) = nil, want rejection", model)
		}
	}
	if err := ValidateModel("   "); err == nil {
		t.Fatal("ValidateModel(empty) = nil, want rejection")
	}
}

func TestResolveModel(t *testing.T) {
	// Explicit request wins.
	if got, err := ResolveModel(ModelPro, ModelFlash); err != nil || got != ModelPro {
		t.Fatalf("ResolveModel(pro, flash) = %q, %v; want pro, nil", got, err)
	}
	// Empty request falls back to the configured default.
	if got, err := ResolveModel("", ModelPro); err != nil || got != ModelPro {
		t.Fatalf("ResolveModel(\"\", pro) = %q, %v; want pro, nil", got, err)
	}
	// Empty request and empty default fall back to the built-in default.
	if got, err := ResolveModel("", ""); err != nil || got != DefaultModel {
		t.Fatalf("ResolveModel(\"\", \"\") = %q, %v; want %q, nil", got, err, DefaultModel)
	}
	// A stale runtime override cannot route to a retired family.
	if _, err := ResolveModel("mimo-v2.5-flash", ""); err == nil {
		t.Fatal("ResolveModel accepted a retired model")
	}
	if _, err := ResolveModel("", "mimo-v2.5-pro"); err == nil {
		t.Fatal("ResolveModel accepted a retired default model")
	}
}
