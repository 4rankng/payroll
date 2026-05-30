package transactions

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestFSM_ValidAndInvalidTransitions(t *testing.T) {
	t.Parallel()

	type tc struct {
		name    string
		from    State
		trigger Trigger
		wantOK  bool
		wantTo  State
	}

	cases := []tc{
		// pending: verify and reject are legal
		{"pending+verify=verified", StatePending, TriggerVerify, true, StateVerified},
		{"pending+reject=failed", StatePending, TriggerReject, true, StateFailed},
		{"pending+authorise REJECT", StatePending, TriggerAuthorise, false, ""},
		{"pending+ipn_completed REJECT", StatePending, TriggerIPNCompleted, false, ""},
		{"pending+ipn_failed REJECT", StatePending, TriggerIPNFailed, false, ""},

		// verified: authorise and reject are legal
		{"verified+authorise=authorised", StateVerified, TriggerAuthorise, true, StateAuthorised},
		{"verified+reject=failed", StateVerified, TriggerReject, true, StateFailed},
		{"verified+verify REJECT", StateVerified, TriggerVerify, false, ""},
		{"verified+ipn_completed REJECT", StateVerified, TriggerIPNCompleted, false, ""},

		// authorised: ipn triggers are legal
		{"authorised+ipn_completed=completed", StateAuthorised, TriggerIPNCompleted, true, StateCompleted},
		{"authorised+ipn_failed=failed", StateAuthorised, TriggerIPNFailed, true, StateFailed},
		{"authorised+authorise REJECT", StateAuthorised, TriggerAuthorise, false, ""},
		{"authorised+reject REJECT", StateAuthorised, TriggerReject, false, ""},
		{"authorised+ipn_reversed=reversed", StateAuthorised, TriggerIPNReversed, true, StateReversed},

		// completed: terminal, all rejected
		{"completed+authorise REJECT", StateCompleted, TriggerAuthorise, false, ""},
		{"completed+reject REJECT", StateCompleted, TriggerReject, false, ""},
		{"completed+ipn_completed REJECT", StateCompleted, TriggerIPNCompleted, false, ""},
		{"completed+ipn_reversed=reversed", StateCompleted, TriggerIPNReversed, true, StateReversed},

		// failed: terminal, all rejected
		{"failed+authorise REJECT", StateFailed, TriggerAuthorise, false, ""},
		{"failed+reject REJECT", StateFailed, TriggerReject, false, ""},
		{"failed+ipn_completed REJECT", StateFailed, TriggerIPNCompleted, false, ""},

		// reversed: terminal, all rejected
		{"reversed+authorise REJECT", StateReversed, TriggerAuthorise, false, ""},
		{"reversed+reject REJECT", StateReversed, TriggerReject, false, ""},
		{"reversed+ipn_completed REJECT", StateReversed, TriggerIPNCompleted, false, ""},
		{"reversed+ipn_reversed REJECT", StateReversed, TriggerIPNReversed, false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sm := Build(tc.from, Hooks{})
			err := sm.FireCtx(context.Background(), tc.trigger)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				got, _ := sm.State(context.Background())
				if got != tc.wantTo {
					t.Fatalf("expected destination %q, got %q", tc.wantTo, got)
				}
			} else {
				if err == nil {
					t.Fatal("expected rejection, got success")
				}
			}
		})
	}
}

func TestFSM_Hooks(t *testing.T) {
	t.Parallel()

	t.Run("OnEnterCompleted fires on authorised→completed", func(t *testing.T) {
		var called atomic.Int32
		hooks := Hooks{
			OnEnterCompleted: func(ctx context.Context, _, _ State, _ Trigger) error {
				called.Add(1)
				return nil
			},
		}
		sm := Build(StateAuthorised, hooks)
		if err := sm.FireCtx(context.Background(), TriggerIPNCompleted); err != nil {
			t.Fatal(err)
		}
		if called.Load() != 1 {
			t.Fatalf("expected OnEnterCompleted to fire once, got %d", called.Load())
		}
	})

	t.Run("OnEnterFailed fires on authorised→failed", func(t *testing.T) {
		var called atomic.Int32
		hooks := Hooks{
			OnEnterFailed: func(ctx context.Context, _, _ State, _ Trigger) error {
				called.Add(1)
				return nil
			},
		}
		sm := Build(StateAuthorised, hooks)
		if err := sm.FireCtx(context.Background(), TriggerIPNFailed); err != nil {
			t.Fatal(err)
		}
		if called.Load() != 1 {
			t.Fatalf("expected OnEnterFailed to fire once, got %d", called.Load())
		}
	})

	t.Run("OnEnterReversed fires on completed→reversed", func(t *testing.T) {
		var called atomic.Int32
		hooks := Hooks{
			OnEnterReversed: func(ctx context.Context, _, _ State, _ Trigger) error {
				called.Add(1)
				return nil
			},
		}
		sm := Build(StateCompleted, hooks)
		if err := sm.FireCtx(context.Background(), TriggerIPNReversed); err != nil {
			t.Fatal(err)
		}
		if called.Load() != 1 {
			t.Fatalf("expected OnEnterReversed to fire once, got %d", called.Load())
		}
	})
}

func TestFSM_ConcurrentTransitions(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm := Build(StatePending, Hooks{})
			_ = sm.FireCtx(context.Background(), TriggerVerify)
		}()
	}
	wg.Wait()
}

func TestIsTerminal(t *testing.T) {
	if IsTerminal(StatePending) {
		t.Error("pending should not be terminal")
	}
	if IsTerminal(StateVerified) {
		t.Error("verified should not be terminal")
	}
	if IsTerminal(StateAuthorised) {
		t.Error("authorised should not be terminal")
	}
	if !IsTerminal(StateCompleted) {
		t.Error("completed should be terminal")
	}
	if !IsTerminal(StateFailed) {
		t.Error("failed should be terminal")
	}
	if !IsTerminal(StateReversed) {
		t.Error("reversed should be terminal")
	}
}

func TestCanFire(t *testing.T) {
	if !CanFire(StatePending, TriggerVerify) {
		t.Error("pending should be able to fire verify")
	}
	if !CanFire(StatePending, TriggerReject) {
		t.Error("pending should be able to fire reject")
	}
	if CanFire(StatePending, TriggerAuthorise) {
		t.Error("pending should NOT be able to fire authorise (must go through verify first)")
	}
	if !CanFire(StateVerified, TriggerAuthorise) {
		t.Error("verified should be able to fire authorise")
	}
	if !CanFire(StateVerified, TriggerReject) {
		t.Error("verified should be able to fire reject")
	}
	if !CanFire(StateAuthorised, TriggerIPNCompleted) {
		t.Error("authorised should be able to fire ipn_completed")
	}
	if CanFire(StateCompleted, TriggerAuthorise) {
		t.Error("completed should not be able to fire authorise")
	}
	if CanFire(StateFailed, TriggerIPNCompleted) {
		t.Error("failed should not be able to fire ipn_completed")
	}
	if !CanFire(StateAuthorised, TriggerIPNReversed) {
		t.Error("authorised should be able to fire ipn_reversed")
	}
	if !CanFire(StateCompleted, TriggerIPNReversed) {
		t.Error("completed should be able to fire ipn_reversed")
	}
	if CanFire(StateReversed, TriggerIPNCompleted) {
		t.Error("reversed should not be able to fire ipn_completed")
	}
}
