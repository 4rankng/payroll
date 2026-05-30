package disbursement_test

import (
	"context"
	"errors"
	"testing"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain/ports/infrastructure"
)

type stubProvider struct{ name string }

func (s stubProvider) Name() string { return s.name }
func (s stubProvider) InitiateTransfer(context.Context, infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	return nil, nil
}
func (s stubProvider) VerifyAndParseWebhook(context.Context, map[string]any) (*infrastructure.WebhookEvent, error) {
	return nil, nil
}

func TestRegistry_Active_EmptyReturnsErr(t *testing.T) {
	r := disbursement.NewRegistry()
	_, err := r.Active(context.Background())
	if !errors.Is(err, disbursement.ErrNoActiveProvider) {
		t.Fatalf("err = %v, want ErrNoActiveProvider", err)
	}
}

func TestRegistry_Active_ReturnsRegisteredProvider(t *testing.T) {
	ninepay := stubProvider{name: "9pay"}
	r := disbursement.NewRegistry(ninepay)

	got, err := r.Active(context.Background())
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	if got.Name() != "9pay" {
		t.Fatalf("Name() = %q, want 9pay", got.Name())
	}
}

func TestRegistry_Active_RegistrationOrderIsDeterministic(t *testing.T) {
	first := stubProvider{name: "9pay"}
	second := stubProvider{name: "payos"}
	r := disbursement.NewRegistry(first, second)

	got, err := r.Active(context.Background())
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	if got.Name() != "9pay" {
		t.Fatalf("Name() = %q, want 9pay (first registered)", got.Name())
	}
}

func TestRegistry_ByNameAndNames(t *testing.T) {
	ninepay := stubProvider{name: "9pay"}
	r := disbursement.NewRegistry(ninepay)

	if _, ok := r.ByName("9pay"); !ok {
		t.Fatalf("ByName(\"9pay\") missing")
	}
	if _, ok := r.ByName("payos"); ok {
		t.Fatalf("ByName(\"payos\") should be absent")
	}
	if got := r.Names(); len(got) != 1 || got[0] != "9pay" {
		t.Fatalf("Names() = %v, want [9pay]", got)
	}
}

func TestRegistry_RegisterDuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on duplicate Register")
		}
	}()
	r := disbursement.NewRegistry(stubProvider{name: "9pay"})
	r.Register(stubProvider{name: "9pay"})
}

func TestRegistry_RegisterEmptyNamePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for empty Name()")
		}
	}()
	disbursement.NewRegistry(stubProvider{name: ""})
}
