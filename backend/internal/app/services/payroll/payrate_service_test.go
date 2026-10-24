package payroll

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

type attachRepoStub struct {
	domain.PayrateRepository
	timeline []*domain.Payrate
}

func (s *attachRepoStub) GetByProject(context.Context, uint) ([]*domain.Payrate, error) {
	return s.timeline, nil
}

func d(y, m, day int) time.Time {
	return time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.Local)
}

// AttachDisplayEndDates derives each row's end from its successor regardless
// of the repository's sort order: a stored to_date is stale by design under
// the from-date-only model, and a wrong derived end misrepresents history.
func TestPayrateServiceAttachDisplayEndDates(t *testing.T) {
	row1 := &domain.Payrate{ID: 1, FromDate: d(2026, 10, 1)}
	row2 := &domain.Payrate{ID: 2, FromDate: d(2026, 10, 15)}
	row3 := &domain.Payrate{ID: 3, FromDate: d(2026, 11, 1)}

	t.Run("repository returns from_date DESC", func(t *testing.T) {
		svc := &PayrateService{PayrateRepo: &attachRepoStub{timeline: []*domain.Payrate{row3, row2, row1}}}
		require.NoError(t, svc.AttachDisplayEndDates(context.Background(), []*domain.Payrate{row1, row3}))

		require.NotNil(t, row1.ToDate)
		require.True(t, row1.ToDate.Equal(d(2026, 10, 14)))
		require.Nil(t, row3.ToDate, "latest config is open-ended")
	})

	t.Run("repository returns from_date ASC", func(t *testing.T) {
		row1.ToDate, row2.ToDate, row3.ToDate = nil, nil, nil
		svc := &PayrateService{PayrateRepo: &attachRepoStub{timeline: []*domain.Payrate{row1, row2, row3}}}
		require.NoError(t, svc.AttachDisplayEndDates(context.Background(), []*domain.Payrate{row2}))

		require.NotNil(t, row2.ToDate)
		require.True(t, row2.ToDate.Equal(d(2026, 10, 31)))
	})
}
