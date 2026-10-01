package wallet

import "time"

// BalanceAlertState is the single-row persisted marker for the wallet
// low-balance alert. It records which side of the configured threshold the
// balance was on at the last evaluation, so the alert fires once per
// downward crossing (above → below) and re-arms after recovery. A missing
// row means the balance has never been evaluated and counts as "above".
type BalanceAlertState struct {
	ID             uint64
	IsBelow        bool
	LastNotifiedAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
