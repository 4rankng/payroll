package transactions

import (
	"context"
	"fmt"

	"github.com/qmuntal/stateless"
)

// State is the lifecycle status of a wallet payment.
//
// Pending is the initial state on insert. Verified means the account check
// passed. Authorised means the transfer was accepted by the provider.
// Completed/Failed are terminal.
type State string

const (
	StatePending    State = "pending"
	StateVerified   State = "verified"
	StateAuthorised State = "authorised"
	StateCompleted  State = "completed"
	StateFailed     State = "failed"
	StateReversed   State = "reversed"
)

// Trigger is the input that drives the FSM.
type Trigger string

const (
	// TriggerVerify: account check passed.
	TriggerVerify Trigger = "verify"
	// TriggerAuthorise: provider accepted the transfer request.
	TriggerAuthorise Trigger = "authorise"
	// TriggerReject: account check failed or provider rejected synchronously.
	TriggerReject Trigger = "reject"
	// TriggerIPNCompleted: webhook says the bank settled the transfer.
	TriggerIPNCompleted Trigger = "ipn_completed"
	// TriggerIPNFailed: webhook says the bank rejected the transfer.
	TriggerIPNFailed Trigger = "ipn_failed"
	// TriggerIPNReversed: webhook says the bank returned the money after settlement.
	TriggerIPNReversed Trigger = "ipn_reversed"
)

// IsTerminal reports whether s is a final state.
func IsTerminal(s State) bool {
	switch s {
	case StateCompleted, StateFailed, StateReversed:
		return true
	}
	return false
}

// OnEntryFunc is invoked by the FSM after a successful transition into
// the named state.
type OnEntryFunc func(ctx context.Context, from, to State, trigger Trigger) error

// Hooks wires service-layer callbacks into the FSM.
type Hooks struct {
	OnEnterCompleted OnEntryFunc
	OnEnterFailed    OnEntryFunc
	OnEnterReversed  OnEntryFunc
}

// Build constructs a stateless.StateMachine pinned to the given current
// state. The transition table:
//
//	pending    --verify-->        verified
//	pending    --reject-->        failed
//	verified   --authorise-->     authorised
//	verified   --reject-->        failed
//	authorised --ipn_completed--> completed
//	authorised --ipn_failed-->    failed
//	authorised --ipn_reversed-->  reversed
//	completed  --ipn_reversed-->  reversed
func Build(current State, hooks Hooks) *stateless.StateMachine {
	sm := stateless.NewStateMachine(current)

	sm.Configure(StatePending).
		Permit(TriggerVerify, StateVerified).
		Permit(TriggerReject, StateFailed)

	sm.Configure(StateVerified).
		Permit(TriggerAuthorise, StateAuthorised).
		Permit(TriggerReject, StateFailed)

	sm.Configure(StateAuthorised).
		Permit(TriggerIPNCompleted, StateCompleted).
		Permit(TriggerIPNFailed, StateFailed).
		Permit(TriggerIPNReversed, StateReversed)

	sm.Configure(StateCompleted).
		Permit(TriggerIPNReversed, StateReversed)

	completed := sm.Configure(StateCompleted) // re-configure merges with the above
	if hooks.OnEnterCompleted != nil {
		completed.OnEntry(func(ctx context.Context, args ...any) error {
			from, to, trig := transitionStates(args)
			return hooks.OnEnterCompleted(ctx, from, to, trig)
		})
	}

	failed := sm.Configure(StateFailed)
	if hooks.OnEnterFailed != nil {
		failed.OnEntry(func(ctx context.Context, args ...any) error {
			from, to, trig := transitionStates(args)
			return hooks.OnEnterFailed(ctx, from, to, trig)
		})
	}

	reversed := sm.Configure(StateReversed)
	if hooks.OnEnterReversed != nil {
		reversed.OnEntry(func(ctx context.Context, args ...any) error {
			from, to, trig := transitionStates(args)
			return hooks.OnEnterReversed(ctx, from, to, trig)
		})
	}

	return sm
}

func transitionStates(args []any) (State, State, Trigger) {
	if len(args) == 0 {
		return "", "", ""
	}
	tr, ok := args[0].(stateless.Transition)
	if !ok {
		return "", "", ""
	}
	from, _ := tr.Source.(State)
	to, _ := tr.Destination.(State)
	trig, _ := tr.Trigger.(Trigger)
	return from, to, trig
}

// CanFire reports whether the given trigger would be accepted from the
// current state.
func CanFire(current State, trigger Trigger) bool {
	sm := Build(current, Hooks{})
	ok, _ := sm.CanFire(trigger)
	return ok
}

// TerminalStateError describes a trigger fired against a state that
// rejects it.
type TerminalStateError struct {
	State   State
	Trigger Trigger
}

func (e TerminalStateError) Error() string {
	return fmt.Sprintf("transactions: cannot fire trigger %q from state %q", e.Trigger, e.State)
}
