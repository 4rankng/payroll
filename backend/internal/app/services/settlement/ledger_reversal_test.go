package settlement

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// A reversal mirrors a whole balanced block. The tests below pin the three
// properties the endpoint is expected to guarantee, because getting any of them
// wrong silently corrupts the books rather than failing loudly:
//
//  1. the mirrors of the block balance, so SUM(debit) == SUM(credit) survives;
//  2. every mirror points at the entry it offsets, so "already reversed?" is a
//     lookup instead of a guess;
//  3. a reversal cannot be repeated, applied to a mirror, or applied to a group
//     that is not itself balanced.
type reversalRepoStub struct {
	domain.LedgerEntryRepository // embedded: only the methods this path needs are real

	entries    []*domain.LedgerEntry
	created    []*domain.LedgerEntry
	failCreate error
}

func (r *reversalRepoStub) GetByID(_ context.Context, id uint) (*domain.LedgerEntry, error) {
	for _, e := range r.entries {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, domain.NewNotFoundError("Không tìm thấy bản ghi sổ cái")
}

func (r *reversalRepoStub) ListReversalGroup(_ context.Context, entry *domain.LedgerEntry) ([]*domain.LedgerEntry, error) {
	group := make([]*domain.LedgerEntry, 0, len(r.entries))
	for _, e := range r.entries {
		if e.CreatedAt.Equal(entry.CreatedAt) && e.CreatedBy == entry.CreatedBy {
			group = append(group, e)
		}
	}
	return group, nil
}

func (r *reversalRepoStub) HasReversal(_ context.Context, entryID uint) (bool, error) {
	for _, e := range r.created {
		if e.ReversalOfEntryID != nil && *e.ReversalOfEntryID == entryID {
			return true, nil
		}
	}
	return false, nil
}

func (r *reversalRepoStub) CreateTransaction(_ context.Context, entries []*domain.LedgerEntry) error {
	if r.failCreate != nil {
		return r.failCreate
	}
	// Mirror the real invariant: a block that does not balance is refused.
	var debit, credit int64
	for _, e := range entries {
		debit += e.Debit
		credit += e.Credit
	}
	if debit != credit {
		return domain.NewValidationError("bút toán không cân bằng")
	}
	for i, e := range entries {
		e.ID = uint(1000 + len(r.created) + i)
		r.created = append(r.created, e)
	}
	return nil
}

type recordingEventBus struct{ published int }

func (b *recordingEventBus) Publish(context.Context, ...domain.DomainEvent) error {
	b.published++
	return nil
}
func (b *recordingEventBus) Subscribe(string, domain.EventHandler) {}
func (b *recordingEventBus) SubscribeAll(domain.EventHandler)      {}

func newReversalFixture() (*LedgerService, *reversalRepoStub, *recordingEventBus, []*domain.LedgerEntry) {
	written := time.Date(2026, 9, 20, 10, 30, 0, 123000000, time.UTC)
	block := []*domain.LedgerEntry{
		{ID: 1, Date: written, Account: "cash", Party: "VFIC", Debit: 1000000, CreatedBy: 7, CreatedAt: written},
		{ID: 2, Date: written, Account: "payable", Party: "VFIC", Credit: 1000000, CreatedBy: 7, CreatedAt: written},
	}
	repo := &reversalRepoStub{entries: block}
	bus := &recordingEventBus{}
	return NewLedgerService(repo, bus, nil), repo, bus, block
}

func TestReverseEntryMirrorsTheWholeBalancedBlock(t *testing.T) {
	svc, repo, bus, block := newReversalFixture()

	result, err := svc.ReverseEntry(context.Background(), block[0].ID, "giao dịch ghi sai", 9)
	require.NoError(t, err)

	require.Equal(t, 2, result.Entries, "both legs of the block are reversed")
	require.Len(t, repo.created, 2)

	var debit, credit int64
	for _, mirror := range repo.created {
		debit += mirror.Debit
		credit += mirror.Credit
		require.NotNil(t, mirror.ReversalOfEntryID, "every mirror links to the entry it offsets")
		require.Equal(t, "giao dịch ghi sai", mirror.ReversalReason, "the operator's reason is persisted")
		require.Equal(t, uint(9), mirror.CreatedBy)
	}
	require.Equal(t, debit, credit, "the mirrored block balances, so the ledger stays balanced")

	// The caller gets the mirror of the entry it asked about, not an arbitrary leg.
	require.Equal(t, block[0].ID, *result.Reversed.ReversalOfEntryID)
	require.Equal(t, block[0].Credit, result.Reversed.Debit)
	require.Equal(t, block[0].Debit, result.Reversed.Credit)
	require.Equal(t, len(repo.created), bus.published, "every mirror is published")
}

func TestReverseEntryRefusesASecondReversal(t *testing.T) {
	svc, _, _, block := newReversalFixture()

	_, err := svc.ReverseEntry(context.Background(), block[0].ID, "sai sót", 9)
	require.NoError(t, err)

	_, err = svc.ReverseEntry(context.Background(), block[0].ID, "bấm lại", 9)
	require.Error(t, err)
	require.True(t, domain.IsConflictError(err), "a repeat reversal is a conflict, not a new offset: %v", err)
}

func TestReverseEntryRefusesToReverseAReversal(t *testing.T) {
	svc, repo, _, block := newReversalFixture()

	result, err := svc.ReverseEntry(context.Background(), block[0].ID, "sai sót", 9)
	require.NoError(t, err)

	// The mirror is now a live entry; compounding reversals has no agreed meaning
	// in the books, so it is refused with a validation error.
	repo.entries = append(repo.entries, result.Reversed)
	_, err = svc.ReverseEntry(context.Background(), result.Reversed.ID, "đảo lại", 9)
	require.Error(t, err)
	require.True(t, domain.IsValidationError(err), "want validation error, got %v", err)
}

func TestReverseEntryRefusesAnUnbalancedGroup(t *testing.T) {
	svc, repo, _, _ := newReversalFixture()
	// A legacy one-sided entry: no block, nothing to mirror that would balance.
	written := time.Date(2026, 9, 19, 8, 0, 0, 0, time.UTC)
	repo.entries = []*domain.LedgerEntry{
		{ID: 5, Date: written, Account: "cash", Party: "VFIC", Debit: 500000, CreatedBy: 7, CreatedAt: written},
	}

	_, err := svc.ReverseEntry(context.Background(), 5, "lý do", 9)
	require.Error(t, err)
	require.True(t, domain.IsValidationError(err), "want validation error, got %v", err)
	require.Empty(t, repo.created, "nothing is written when the group cannot balance")
}

func TestReverseEntryRequiresAReason(t *testing.T) {
	svc, repo, _, block := newReversalFixture()

	_, err := svc.ReverseEntry(context.Background(), block[0].ID, "   ", 9)
	require.Error(t, err)
	require.True(t, domain.IsValidationError(err), "want validation error, got %v", err)
	require.Empty(t, repo.created)
}
