package observability

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		cfg     LogConfig
		wantErr bool
	}{
		{
			name: "debug level",
			cfg: LogConfig{
				Level: "debug",
				File:  "",
			},
			wantErr: false,
		},
		{
			name: "info level",
			cfg: LogConfig{
				Level: "info",
				File:  "",
			},
			wantErr: false,
		},
		{
			name: "warn level",
			cfg: LogConfig{
				Level: "warn",
				File:  "",
			},
			wantErr: false,
		},
		{
			name: "warning level",
			cfg: LogConfig{
				Level: "warning",
				File:  "",
			},
			wantErr: false,
		},
		{
			name: "error level",
			cfg: LogConfig{
				Level: "error",
				File:  "",
			},
			wantErr: false,
		},
		{
			name: "unknown level defaults to info",
			cfg: LogConfig{
				Level: "unknown",
				File:  "",
			},
			wantErr: false,
		},
		{
			name: "with log file",
			cfg: LogConfig{
				Level: "info",
				File:  filepath.Join(os.TempDir(), "test-log.log"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if logger == nil && !tt.wantErr {
				t.Error("NewLogger() returned nil logger")
			}

			// Clean up log file if created
			if tt.cfg.File != "" {
				_ = os.Remove(tt.cfg.File)
			}
		})
	}
}

func TestGetLogger(t *testing.T) {
	// Reset instance for test
	instance = nil

	logger1 := GetLogger()
	if logger1 == nil {
		t.Error("GetLogger() returned nil")
	}

	// Subsequent calls should return the same instance
	logger2 := GetLogger()
	if logger1 != logger2 {
		t.Error("GetLogger() did not return singleton instance")
	}
}

func TestSetLogger(t *testing.T) {
	// Create a custom logger
	customLogger := slog.Default()

	// Set the custom logger
	SetLogger(customLogger)

	// Verify it was set
	retrieved := GetLogger()
	if retrieved != customLogger {
		t.Error("SetLogger() did not set the logger correctly")
	}

	// Reset for other tests
	instance = nil
}

func TestGetLoggerConcurrency(t *testing.T) {
	// Reset instance for test
	instance = nil

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			logger := GetLogger()
			if logger == nil {
				t.Error("GetLogger() returned nil in concurrent access")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
