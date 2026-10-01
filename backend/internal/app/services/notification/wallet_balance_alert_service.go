package notification

import (
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/domain"
	"api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/utils"
)

type walletBalanceAlertBalanceReader interface {
	GetBalance(ctx context.Context) (*wallet.WalletBalance, error)
}

type walletBalanceAlertStateRepository interface {
	Get(ctx context.Context) (*wallet.BalanceAlertState, error)
	Save(ctx context.Context, state *wallet.BalanceAlertState) error
}

type walletBalanceAlertThresholdSource interface {
	GetWalletBalanceAlertThreshold(ctx context.Context) int64
}

type walletBalanceAlertNotifier interface {
	NotifyUsersByRole(ctx context.Context, role domain.UserRole, notificationType domain.NotificationType, title, message string) error
}

// WalletBalanceAlertService warns every administrator the moment the wallet
// balance crosses below the configured threshold — once per downward
// crossing (above → below). The crossing marker is persisted, so restarts
// never resend while the balance stays low; the alert re-arms silently once
// the balance recovers to the threshold or above. Recovery itself sends no
// notification.
type WalletBalanceAlertService struct {
	balance   walletBalanceAlertBalanceReader
	threshold walletBalanceAlertThresholdSource
	state     walletBalanceAlertStateRepository
	notifier  walletBalanceAlertNotifier
	logger    *slog.Logger
}

func NewWalletBalanceAlertService(
	balance walletBalanceAlertBalanceReader,
	threshold walletBalanceAlertThresholdSource,
	state walletBalanceAlertStateRepository,
	notifier walletBalanceAlertNotifier,
	logger *slog.Logger,
) *WalletBalanceAlertService {
	if logger == nil {
		logger = slog.Default()
	}
	return &WalletBalanceAlertService{
		balance:   balance,
		threshold: threshold,
		state:     state,
		notifier:  notifier,
		logger:    logger,
	}
}

// Check evaluates the current wallet balance against the configured threshold
// and notifies every admin on a downward crossing (above → below). It returns
// whether an alert notification was sent on this run.
func (s *WalletBalanceAlertService) Check(ctx context.Context) (bool, error) {
	balance, err := s.balance.GetBalance(ctx)
	if err != nil {
		return false, fmt.Errorf("wallet balance alert: read balance: %w", err)
	}

	threshold := s.threshold.GetWalletBalanceAlertThreshold(ctx)
	below := balance.Available < threshold

	state, err := s.state.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("wallet balance alert: read state: %w", err)
	}
	if state == nil {
		state = &wallet.BalanceAlertState{}
	}

	if below == state.IsBelow {
		// Same side of the threshold as the last evaluation: nothing to do.
		return false, nil
	}

	if !below {
		// Recovery: re-arm silently. No notification on upward crossings.
		state.IsBelow = false
		if err := s.state.Save(ctx, state); err != nil {
			return false, fmt.Errorf("wallet balance alert: save state: %w", err)
		}
		s.logger.Info("wallet balance recovered above alert threshold",
			"available", balance.Available,
			"threshold", threshold,
		)
		return false, nil
	}

	// Downward crossing. Send first, persist after — a failed send leaves the
	// state unadvanced so the next run retries instead of silently dropping
	// the alert. A send that succeeds but fails to persist may notify again
	// on the next run; a duplicated alert beats a missing one.
	title := "Ví tiền dưới ngưỡng cảnh báo"
	message := fmt.Sprintf(
		"Số dư ví tiền hiện còn %s, thấp hơn ngưỡng cảnh báo %s. Vui lòng nạp thêm tiền vào ví để tiếp tục chi trả ứng lương.",
		utils.FormatVND(balance.Available),
		utils.FormatVND(threshold),
	)
	if err := s.notifier.NotifyUsersByRole(ctx, domain.RoleAdmin, domain.NotificationTypeWalletBalanceLow, title, message); err != nil {
		return false, fmt.Errorf("wallet balance alert: notify admins: %w", err)
	}

	state.IsBelow = true
	now := clock.Now()
	state.LastNotifiedAt = &now
	if err := s.state.Save(ctx, state); err != nil {
		return true, fmt.Errorf("wallet balance alert: save state: %w", err)
	}
	s.logger.Warn("wallet balance below alert threshold",
		"available", balance.Available,
		"threshold", threshold,
	)
	return true, nil
}
