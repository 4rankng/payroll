package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.InitialDelay)
	assert.Equal(t, 5*time.Second, cfg.MaxDelay)
	assert.Equal(t, 2.0, cfg.BackoffFactor)
}

func TestDefaultDatabaseConfig(t *testing.T) {
	cfg := DefaultDatabaseConfig()

	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, 50*time.Millisecond, cfg.InitialDelay)
	assert.Equal(t, 10*time.Second, cfg.MaxDelay)
	assert.Equal(t, 1.5, cfg.BackoffFactor)
	assert.Equal(t, 30*time.Second, cfg.OperationTimeout)
}

func TestIsDatabaseConnectionError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{
			name:    "Nil error",
			err:     nil,
			wantErr: false,
		},
		{
			name:    "Driver bad connection",
			err:     errors.New("driver: bad connection"),
			wantErr: true,
		},
		{
			name:    "Connection refused",
			err:     errors.New("connection refused"),
			wantErr: true,
		},
		{
			name:    "Connection reset",
			err:     errors.New("connection reset"),
			wantErr: true,
		},
		{
			name:    "Connection timeout",
			err:     errors.New("connection timeout"),
			wantErr: true,
		},
		{
			name:    "Broken pipe",
			err:     errors.New("broken pipe"),
			wantErr: true,
		},
		{
			name:    "Server has gone away",
			err:     errors.New("mysql: server has gone away"),
			wantErr: true,
		},
		{
			name:    "Database is locked",
			err:     errors.New("database is locked"),
			wantErr: true,
		},
		{
			name:    "Deadlock detected",
			err:     errors.New("deadlock detected"),
			wantErr: true,
		},
		{
			name:    "Non-connection error",
			err:     errors.New("some random error"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDatabaseConnectionError(tt.err)
			assert.Equal(t, tt.wantErr, got)
		})
	}
}

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{
			name:    "Nil error",
			err:     nil,
			wantErr: false,
		},
		{
			name:    "Connection error",
			err:     errors.New("connection refused"),
			wantErr: true,
		},
		{
			name:    "Too many connections",
			err:     errors.New("too many connections"),
			wantErr: true,
		},
		{
			name:    "Max connections reached",
			err:     errors.New("max connections reached"),
			wantErr: true,
		},
		{
			name:    "Temporary failure",
			err:     errors.New("temporary failure in name resolution"),
			wantErr: true,
		},
		{
			name:    "Service unavailable",
			err:     errors.New("service unavailable"),
			wantErr: true,
		},
		{
			name:    "Resource temporarily unavailable",
			err:     errors.New("resource temporarily unavailable"),
			wantErr: true,
		},
		{
			name:    "Non-transient error",
			err:     errors.New("validation error"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTransientError(tt.err)
			assert.Equal(t, tt.wantErr, got)
		})
	}
}

func TestWithExponentialBackoff_Success(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:    3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	callCount := 0
	operation := func() error {
		callCount++
		return nil
	}

	err := WithExponentialBackoff(ctx, cfg, operation)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestWithExponentialBackoff_RetryOnTransientError(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:    2,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 2 {
			return errors.New("connection refused")
		}
		return nil
	}

	err := WithExponentialBackoff(ctx, cfg, operation)
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestWithExponentialBackoff_NonTransientError(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:    3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	callCount := 0
	expectedErr := errors.New("validation error")
	operation := func() error {
		callCount++
		return expectedErr
	}

	err := WithExponentialBackoff(ctx, cfg, operation)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 1, callCount) // Should not retry non-transient errors
}

func TestWithExponentialBackoff_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:    2,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("connection refused")
	}

	err := WithExponentialBackoff(ctx, cfg, operation)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed after multiple retries")
	assert.Equal(t, 3, callCount) // Initial attempt + 2 retries
}

func TestWithExponentialBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{
		MaxRetries:    3,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	callCount := 0
	operation := func() error {
		callCount++
		if callCount == 1 {
			// Cancel context after first attempt
			cancel()
		}
		return errors.New("connection refused")
	}

	err := WithExponentialBackoff(ctx, cfg, operation)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestWithDatabaseRetry_Success(t *testing.T) {
	ctx := context.Background()
	cfg := DatabaseOperationConfig{
		MaxRetries:       3,
		InitialDelay:     10 * time.Millisecond,
		MaxDelay:         100 * time.Millisecond,
		BackoffFactor:    1.5,
		OperationTimeout: 1 * time.Second,
	}

	callCount := 0
	operation := func() error {
		callCount++
		return nil
	}

	err := WithDatabaseRetry(ctx, cfg, operation)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestWithDatabaseRetry_RetryOnTransientError(t *testing.T) {
	ctx := context.Background()
	cfg := DatabaseOperationConfig{
		MaxRetries:       2,
		InitialDelay:     10 * time.Millisecond,
		MaxDelay:         100 * time.Millisecond,
		BackoffFactor:    1.5,
		OperationTimeout: 1 * time.Second,
	}

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 2 {
			return errors.New("connection timeout")
		}
		return nil
	}

	err := WithDatabaseRetry(ctx, cfg, operation)
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestWithDatabaseRetry_OperationTimeout(t *testing.T) {
	ctx := context.Background()
	cfg := DatabaseOperationConfig{
		MaxRetries:       5,
		InitialDelay:     20 * time.Millisecond,
		MaxDelay:         50 * time.Millisecond,
		BackoffFactor:    2.0,
		OperationTimeout: 80 * time.Millisecond, // Short timeout that will be exceeded during retries
	}

	// Operation that always fails with transient error
	// The retries and delays will exceed the timeout
	operation := func() error {
		return errors.New("connection timeout") // Transient error
	}

	err := WithDatabaseRetry(ctx, cfg, operation)
	assert.Error(t, err)
	// Should timeout before exhausting all retries
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestWithDatabaseRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := DatabaseOperationConfig{
		MaxRetries:       3,
		InitialDelay:     10 * time.Millisecond,
		MaxDelay:         100 * time.Millisecond,
		BackoffFactor:    1.5,
		OperationTimeout: 1 * time.Second,
	}

	callCount := 0
	expectedErr := errors.New("constraint violation")
	operation := func() error {
		callCount++
		return expectedErr
	}

	err := WithDatabaseRetry(ctx, cfg, operation)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 1, callCount)
}
