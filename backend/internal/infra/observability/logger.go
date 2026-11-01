package observability

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type LogConfig struct {
	Level string
	Dir   string // directory holding the daily payroll-backend-YYYYMMDD.log files
}

var (
	instance *slog.Logger
	mu       sync.RWMutex
)

// hcmLocation pins log file names to the company's business timezone so the
// YYYYMMDD in the filename matches the Vietnamese calendar day regardless of
// the container's TZ setting.
var hcmLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}()

const (
	dailyFilePrefix   = "payroll-backend"
	dailyFileLayout   = "20060102"
	dailyFileKeepDays = 14
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
		// Fallback for import-time callers (package-level vars) that run
		// before main() loads config: stdout only. Deliberately no file
		// output here — creating it with a guessed directory would claim the
		// shared DailyWriter sync.Once with the wrong dir (PAYR-32 lesson).
		instance = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
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

	// Dual output (PAYR-32): stdout always (so `docker logs` keeps working)
	// plus the persistent daily file on the /app/logs volume when available.
	// If the log directory cannot be created/opened we keep stdout-only
	// logging instead of failing the process.
	writers := []io.Writer{os.Stdout}
	if w := DailyWriter(cfg.Dir); w != nil {
		writers = append(writers, w)
	}
	writer := io.MultiWriter(writers...)

	// Create handler with JSON format
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})

	return slog.New(handler), nil
}

// DailyWriter returns the shared daily-rotating file writer rooted at dir.
// It appends to `<dir>/payroll-backend-YYYYMMDD.log` (day in Asia/Ho_Chi_Minh)
// and prunes rotated files older than 14 days on startup and at every
// rollover. Both the slog app logger and GIN (DefaultWriter/DefaultErrorWriter)
// share this single writer, so access logs and app logs land in the same
// daily file. Returns nil when the directory cannot be created or opened —
// callers should then keep stdout-only logging.
func DailyWriter(dir string) io.Writer {
	dailyOnce.Do(func() {
		if dir == "" {
			dir = "logs"
		}
		dailyShared, dailyErr = newDailyFileWriter(dir, time.Now)
	})
	return dailyShared
}

// DailyWriterError reports why DailyWriter returned nil, for startup diagnostics.
func DailyWriterError() error {
	return dailyErr
}

var (
	dailyOnce   sync.Once
	dailyShared *dailyFileWriter
	dailyErr    error
)

// dailyFileWriter is a concurrency-safe io.Writer that appends to
// <dir>/<prefix>-YYYYMMDD.log, switching files at Ho_Chi_Minh midnight and
// deleting rotated files older than dailyFileKeepDays. File write errors are
// swallowed (best-effort persistent copy): stdout, written before this writer
// in every MultiWriter we build, has already received the line.
type dailyFileWriter struct {
	mu   sync.Mutex
	dir  string
	now  func() time.Time // injectable for tests
	day  string
	file *os.File
}

func newDailyFileWriter(dir string, nowFn func() time.Time) (*dailyFileWriter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir %s: %w", dir, err)
	}
	w := &dailyFileWriter{dir: dir, now: nowFn}
	if err := w.roll(w.dayKey()); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *dailyFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if key := w.dayKey(); key != w.day {
		// Best-effort rollover; if opening the new day's file fails we keep
		// appending to yesterday's file so no line is lost mid-transition.
		_ = w.roll(key)
	}
	if w.file == nil {
		return len(p), nil
	}
	if _, err := w.file.Write(p); err != nil {
		return len(p), nil
	}
	return len(p), nil
}

func (w *dailyFileWriter) dayKey() string {
	return w.now().In(hcmLocation).Format(dailyFileLayout)
}

// roll switches to the file for `key` and prunes stale files. On open failure
// the previous file stays open so writes continue undisturbed.
func (w *dailyFileWriter) roll(key string) error {
	path := filepath.Join(w.dir, dailyFilePrefix+"-"+key+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", path, err)
	}
	if w.file != nil {
		_ = w.file.Close()
	}
	w.file = f
	w.day = key
	w.prune()
	return nil
}

// prune deletes rotated payroll-backend-*.log files older than dailyFileKeepDays.
func (w *dailyFileWriter) prune() {
	// Compare calendar days only: a file written on the keep-boundary day
	// itself must survive regardless of the current time of day.
	boundary := w.now().In(hcmLocation).AddDate(0, 0, -dailyFileKeepDays)
	cutoff := time.Date(boundary.Year(), boundary.Month(), boundary.Day(), 0, 0, 0, 0, hcmLocation)
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}
	prefix := dailyFilePrefix + "-"
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".log") {
			continue
		}
		dayPart := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".log")
		dayTime, err := time.ParseInLocation(dailyFileLayout, dayPart, hcmLocation)
		if err != nil {
			continue // not one of our rotated files
		}
		if dayTime.Before(cutoff) {
			_ = os.Remove(filepath.Join(w.dir, name))
		}
	}
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
