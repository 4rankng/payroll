package observability

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

type LogConfig struct {
	Level string
	File  string
}

var (
	instance *slog.Logger
	mu       sync.RWMutex
)

// GetLogger returns the singleton logger instance with lazy initialization.
func GetLogger() *slog.Logger {
	mu.RLock()
	if instance != nil {
		defer mu.RUnlock()
		return instance
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	if instance == nil {
		// Initialize with default config if not set
		defaultConfig := LogConfig{
			Level: "info",
			File:  "logs/app.log",
		}
		var err error
		instance, err = NewLogger(defaultConfig)
		if err != nil {
			// Fallback to default logger if initialization fails
			instance = slog.Default()
		}
	}
	return instance
}

// SetLogger allows setting a custom logger instance (useful for testing or custom initialization)
func SetLogger(logger *slog.Logger) {
	mu.Lock()
	defer mu.Unlock()
	instance = logger
}

func NewLogger(cfg LogConfig) (*slog.Logger, error) {
	// Parse log level
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create log directory if it doesn't exist
	if cfg.File != "" {
		logDir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}
	}

	// Setup log output
	var writer io.Writer = os.Stdout

	if cfg.File != "" {
		// Use lumberjack for log rotation
		writer = io.MultiWriter(
			os.Stdout,
			&lumberjack.Logger{
				Filename:   cfg.File,
				MaxSize:    100, // MB
				MaxAge:     7,   // days
				MaxBackups: 5,
				LocalTime:  true,
				Compress:   true,
			},
		)
	}

	// Create handler with JSON format
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})

	return slog.New(handler), nil
}

// NewFileLogger creates a slog.Logger that writes JSON to the given file
// with lumberjack rotation. Returns nil if filename is empty.
func NewFileLogger(filename string) (*slog.Logger, error) {
	if filename == "" {
		return nil, nil
	}
	logDir := filepath.Dir(filename)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	writer := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    50,
		MaxAge:     7,
		MaxBackups: 3,
		LocalTime:  true,
		Compress:   true,
	}
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(handler), nil
}
