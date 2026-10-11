package llmchat

import (
	"context"

	"api-server/internal/domain/wallet"
)

// walletSourceAdapter adapts the wallet service (computed balance) to the
// narrow chat seam, so the tool layer depends on one method instead of the
// full WalletService surface.
type walletSourceAdapter struct{ svc wallet.WalletService }

// NewWalletInfoSource builds the adapter. Returns nil for a nil service (the
// admin wallet tool then fails closed with a store error, never a panic).
func NewWalletInfoSource(svc wallet.WalletService) WalletInfoSource {
	if svc == nil {
		return nil
	}
	return walletSourceAdapter{svc: svc}
}

func (a walletSourceAdapter) WalletBalance(ctx context.Context) (available, pendingIn, pendingOut, limbo int64, err error) {
	balance, err := a.svc.GetBalance(ctx)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return balance.Available, balance.PendingIn, balance.PendingOut, balance.Limbo, nil
}
