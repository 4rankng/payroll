package asynq

import (
	"testing"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/wallet_bulk"
	"api-server/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestEnqueuePayrollReportEmailReturnsExistingTaskOnRetry(t *testing.T) {
	redis := miniredis.RunT(t)
	client, err := NewClient(config.AsynqConfig{RedisAddr: redis.Addr()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	request := dto.SendPayrollReportEmailRequest{ReportAtDate: "2026-09-01", Recipients: []string{"qa@example.com"}}
	firstID, duplicate, err := client.EnqueuePayrollReportEmail(request, 1, "request-1")
	require.NoError(t, err)
	require.False(t, duplicate)
	retryID, duplicate, err := client.EnqueuePayrollReportEmail(request, 1, "request-1")
	require.NoError(t, err)
	require.True(t, duplicate)
	require.Equal(t, firstID, retryID)
}

func TestEnqueueDuplicateJobsRemainIdempotent(t *testing.T) {
	redis := miniredis.RunT(t)
	client, err := NewClient(config.AsynqConfig{RedisAddr: redis.Addr()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	for name, enqueue := range map[string]func() error{
		"unique BCC import":     func() error { return client.EnqueueBCCImport(123) },
		"wallet ledger task ID": func() error { return client.EnqueueBookBatchLedger(wallet_bulk.BookLedgerPayload{BatchID: 123}) },
	} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, enqueue())
			require.NoError(t, enqueue())
		})
	}
}
