package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/secret"
)

func strPtr(s string) *string { return &s }

func TestNewSettingResponseMasksProtectedValue(t *testing.T) {
	credential := `{"app_id":"123","secret_key":"s3cr3t-key","refresh_token":"tok"}`
	resp := NewSettingResponse(&domain.Settings{
		ID:        7,
		Key:       secret.ZaloCredentialsKey,
		Value:     strPtr(credential),
		ValueType: domain.ValueTypeJSON,
		UpdatedAt: time.Unix(1700000000, 0),
	})

	if !resp.Masked {
		t.Fatal("protected setting was not marked as masked")
	}
	if !resp.Configured {
		t.Fatal("protected setting with a stored value was not marked as configured")
	}
	if resp.Value != nil {
		t.Fatalf("protected value leaked as %q", *resp.Value)
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(encoded), "s3cr3t-key") || strings.Contains(string(encoded), "tok") {
		t.Fatalf("serialized response still carries the secret: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"configured":true`) {
		t.Fatalf("serialized response does not report configured state: %s", encoded)
	}
}

func TestNewSettingResponseReportsUnsetProtectedValue(t *testing.T) {
	resp := NewSettingResponse(&domain.Settings{
		ID:        8,
		Key:       secret.ZaloCredentialsKey,
		Value:     nil,
		ValueType: domain.ValueTypeJSON,
	})
	if !resp.Masked {
		t.Fatal("protected setting was not marked as masked")
	}
	if resp.Configured {
		t.Fatal("protected setting without a value reported itself as configured")
	}
}

func TestNewSettingResponsePassesOrdinaryValuesThrough(t *testing.T) {
	resp := NewSettingResponse(&domain.Settings{
		ID:        9,
		Key:       "transfer_bank_visible",
		Value:     strPtr("true"),
		ValueType: domain.ValueTypeString,
	})
	if resp.Masked {
		t.Fatal("ordinary setting was masked")
	}
	if resp.Value == nil || *resp.Value != "true" {
		t.Fatalf("ordinary value = %v, want true", resp.Value)
	}
	if resp.ValueType != "string" || resp.ID != 9 {
		t.Fatalf("response = %+v, want the setting's id and value type", resp)
	}
}
