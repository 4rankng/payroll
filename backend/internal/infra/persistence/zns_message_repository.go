package persistence

import (
	"context"
	"time"

	"api-server/internal/domain"
)

type ZnsMessageRepository struct{ *BaseRepository }

func NewZnsMessageRepository(db *Database) domain.ZnsMessageRepository {
	return &ZnsMessageRepository{BaseRepository: NewBaseRepository(db)}
}

func (r *ZnsMessageRepository) Record(ctx context.Context, msg *domain.ZnsMessage) error {
	if err := r.dbForContext(ctx).Create(msg).Error; err != nil {
		return domain.NewInternalError("failed to record zns message", err)
	}
	return nil
}

// ListAdmin returns one page of zns_messages newest-first with the total
// matching count. Filters AND together; Query matches phone or provider
// message id. FromDate/ToDate bound created_at inclusively at day grain
// (ToDate is treated as the end of its day).
func (r *ZnsMessageRepository) ListAdmin(ctx context.Context, f domain.ZnsMessageFilter) ([]*domain.ZnsMessage, int64, error) {
	q := r.dbForContext(ctx).Model(&domain.ZnsMessage{})

	if len(f.Statuses) > 0 {
		q = q.Where("status IN ?", f.Statuses)
	}
	if f.Feature != "" {
		q = q.Where("feature = ?", f.Feature)
	}
	if f.TemplateID != "" {
		q = q.Where("template_id = ?", f.TemplateID)
	}
	if f.FromDate != nil {
		q = q.Where("created_at >= ?", dayStart(*f.FromDate))
	}
	if f.ToDate != nil {
		q = q.Where("created_at < ?", dayStart(*f.ToDate).Add(24*time.Hour))
	}
	if f.Query != "" {
		// INSTR (not LIKE) for substring semantics on both MySQL and SQLite —
		// no wildcard escaping to reason about, and a leading-wildcard LIKE
		// cannot use an index anyway.
		q = q.Where("INSTR(phone, ?) > 0 OR INSTR(provider_msg_id, ?) > 0", f.Query, f.Query)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, domain.NewInternalError("failed to count zns messages", err)
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}

	var rows []*domain.ZnsMessage
	err := q.Order("created_at DESC").
		Offset((f.Page - 1) * f.PageSize).
		Limit(f.PageSize).
		Find(&rows).Error
	if err != nil {
		return nil, 0, domain.NewInternalError("failed to list zns messages", err)
	}
	return rows, total, nil
}

func dayStart(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}
