package notification

import (
	"context"
	"errors"
	"testing"

	"api-server/internal/domain"
	"api-server/internal/domain/wallet"

	"github.com/stretchr/testify/require"
)

type fakeWalletAlertBalance struct {
	available int64
	err       error
}

func (f *fakeWalletAlertBalance) GetBalance(ctx context.Context) (*wallet.WalletBalance, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &wallet.WalletBalance{Available: f.available, Currency: "VND"}, nil
}

type fakeWalletAlertState struct {
	stored    *wallet.BalanceAlertState
	getErr    error
	saveErr   error
	saveCount int
}

func (f *fakeWalletAlertState) Get(ctx context.Context) (*wallet.BalanceAlertState, error) {
	return f.stored, f.getErr
}

func (f *fakeWalletAlertState) Save(ctx context.Context, state *wallet.BalanceAlertState) error {
	f.saveCount++
	if f.saveErr != nil {
		return f.saveErr
	}
	cp := *state
	f.stored = &cp
	return nil
}

type fakeWalletAlertThreshold struct {
	value int64
}

func (f *fakeWalletAlertThreshold) GetWalletBalanceAlertThreshold(ctx context.Context) int64 {
	return f.value
}

type walletAlertCall struct {
	role    domain.UserRole
	nType   domain.NotificationType
	title   string
	message string
}

type fakeWalletAlertNotifier struct {
	err   error
	calls []walletAlertCall
}

func (f *fakeWalletAlertNotifier) NotifyUsersByRole(ctx context.Context, role domain.UserRole, notificationType domain.NotificationType, title, message string) error {
	record := walletAlertCall{
		role:    role,
		nType:   notificationType,
		title:   title,
		message: message,
	}
	f.calls = append(f.calls, record)
	return f.err
}

// TestWalletAlertNotifiesOncePerDownwardCrossing walks the balance through a
// full above → below → below → above → below cycle and asserts the alert
// fires on each downward crossing only.
func TestWalletAlertNotifiesOncePerDownwardCrossing(t *testing.T) {
	ctx := context.Background()
	threshold := &fakeWalletAlertThreshold{value: 50_000_000}
	stateRepo := &fakeWalletAlertState{}
	balance := &fakeWalletAlertBalance{available: 107_224_395}
	notifier := &fakeWalletAlertNotifier{}
	svc := NewWalletBalanceAlertService(balance, threshold, stateRepo, notifier, nil)

	// Above threshold: quiet, nothing persisted.
	sent, err := svc.Check(ctx)
	require.NoError(t, err)
	require.False(t, sent)
	require.Empty(t, notifier.calls)
	require.Zero(t, stateRepo.saveCount)

	// First downward crossing: one notification, marker persisted.
	balance.available = 30_000_000
	sent, err = svc.Check(ctx)
	require.NoError(t, err)
	require.True(t, sent)
	require.Len(t, notifier.calls, 1)
	require.Equal(t, domain.RoleAdmin, notifier.calls[0].role)
	require.Equal(t, domain.NotificationTypeWalletBalanceLow, notifier.calls[0].nType)
	require.Contains(t, notifier.calls[0].message, "30.000.000 ₫")
	require.Contains(t, notifier.calls[0].message, "50.000.000 ₫")
	require.True(t, stateRepo.stored.IsBelow)
	require.NotNil(t, stateRepo.stored.LastNotifiedAt)

	// Still below (deeper): quiet, no save churn.
	balance.available = 29_000_000
	sent, err = svc.Check(ctx)
	require.NoError(t, err)
	require.False(t, sent)
	require.Len(t, notifier.calls, 1)
	require.Equal(t, 1, stateRepo.saveCount)

	// Recovery above: silent re-arm, no notification.
	balance.available = 80_000_000
	sent, err = svc.Check(ctx)
	require.NoError(t, err)
	require.False(t, sent)
	require.Len(t, notifier.calls, 1)
	require.False(t, stateRepo.stored.IsBelow)
	require.Equal(t, 2, stateRepo.saveCount)

	// Second downward crossing: notifies again.
	balance.available = 20_000_000
	sent, err = svc.Check(ctx)
	require.NoError(t, err)
	require.True(t, sent)
	require.Len(t, notifier.calls, 2)
	require.Equal(t, 3, stateRepo.saveCount)
}

// TestWalletAlertNotifyFailureLeavesStateUnadvanced verifies a failed send
// does not advance the marker, so the next run retries instead of dropping
// the alert.
func TestWalletAlertNotifyFailureLeavesStateUnadvanced(t *testing.T) {
	ctx := context.Background()
	threshold := &fakeWalletAlertThreshold{value: 50_000_000}
	stateRepo := &fakeWalletAlertState{}
	balance := &fakeWalletAlertBalance{available: 40_000_000}
	notifier := &fakeWalletAlertNotifier{err: errors.New("push outage")}
	svc := NewWalletBalanceAlertService(balance, threshold, stateRepo, notifier, nil)

	// Failing send: error surfaced, marker not advanced.
	sent, err := svc.Check(ctx)
	require.Error(t, err)
	require.False(t, sent)
	require.Zero(t, stateRepo.saveCount)

	// Next run with a healthy notifier retries and persists.
	notifier.err = nil
	sent, err = svc.Check(ctx)
	require.NoError(t, err)
	require.True(t, sent)
	require.Len(t, notifier.calls, 2)
	require.True(t, stateRepo.stored.IsBelow)
}

// TestWalletAlertMissingStateRowCountsAsAbove verifies a missing state row
// counts as "above", so the first downward crossing after deploy notifies.
func TestWalletAlertMissingStateRowCountsAsAbove(t *testing.T) {
	ctx := context.Background()
	// stored stays nil — the service nil-guards to a zero state.
	stateRepo := &fakeWalletAlertState{}
	threshold := &fakeWalletAlertThreshold{value: 50_000_000}
	balance := &fakeWalletAlertBalance{available: 30_000_000}
	notifier := &fakeWalletAlertNotifier{}
	svc := NewWalletBalanceAlertService(balance, threshold, stateRepo, notifier, nil)

	sent, err := svc.Check(ctx)
	require.NoError(t, err)
	require.True(t, sent)
	require.Len(t, notifier.calls, 1)
	require.True(t, stateRepo.stored.IsBelow)
}

// TestWalletAlertBalanceReadErrorPropagates verifies a balance read failure
// aborts the check without touching state or notifications.
func TestWalletAlertBalanceReadErrorPropagates(t *testing.T) {
	ctx := context.Background()
	threshold := &fakeWalletAlertThreshold{value: 50_000_000}
	stateRepo := &fakeWalletAlertState{}
	balance := &fakeWalletAlertBalance{err: errors.New("db down")}
	notifier := &fakeWalletAlertNotifier{}
	svc := NewWalletBalanceAlertService(balance, threshold, stateRepo, notifier, nil)

	sent, err := svc.Check(ctx)
	require.Error(t, err)
	require.False(t, sent)
	require.ErrorContains(t, err, "read balance")
	require.Empty(t, notifier.calls)
	require.Zero(t, stateRepo.saveCount)
}
