package observability

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func fixedNow(t *testing.T, at time.Time) func() time.Time {
	t.Helper()
	return func() time.Time { return at }
}

func dailyName(t *testing.T, at time.Time) string {
	t.Helper()
	return dailyFilePrefix + "-" + at.In(hcmLocation).Format(dailyFileLayout) + ".log"
}

func TestDailyFileWriter_WritesDailyFile(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 1, 10, 30, 0, 0, hcmLocation)
	w, err := newDailyFileWriter(dir, fixedNow(t, now))
	if err != nil {
		t.Fatalf("newDailyFileWriter: %v", err)
	}

	line := "[GIN] hello PAYR-32\n"
	if n, werr := w.Write([]byte(line)); werr != nil || n != len(line) {
		t.Fatalf("write: n=%d err=%v", n, werr)
	}

	path := filepath.Join(dir, dailyName(t, now))
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != line {
		t.Fatalf("file content = %q, want %q", got, line)
	}
}

func TestDailyFileWriter_RollsAtDayChange(t *testing.T) {
	dir := t.TempDir()
	current := time.Date(2026, 11, 1, 23, 59, 0, 0, hcmLocation)
	w, err := newDailyFileWriter(dir, fixedNow(t, current))
	if err != nil {
		t.Fatalf("newDailyFileWriter: %v", err)
	}

	_, _ = w.Write([]byte("day one\n"))
	current = current.Add(24 * time.Hour) // cross midnight into 02/11
	w.now = fixedNow(t, current)
	_, _ = w.Write([]byte("day two\n"))

	day1, err := os.ReadFile(filepath.Join(dir, dailyName(t, time.Date(2026, 11, 1, 0, 0, 0, 0, hcmLocation))))
	if err != nil || !strings.Contains(string(day1), "day one") || strings.Contains(string(day1), "day two") {
		t.Fatalf("day1 file wrong (err=%v content=%q)", err, day1)
	}
	day2, err := os.ReadFile(filepath.Join(dir, dailyName(t, time.Date(2026, 11, 2, 0, 0, 0, 0, hcmLocation))))
	if err != nil || !strings.Contains(string(day2), "day two") || strings.Contains(string(day2), "day one") {
		t.Fatalf("day2 file wrong (err=%v content=%q)", err, day2)
	}
}

func TestDailyFileWriter_PrunesFilesOlderThan14Days(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 1, 8, 0, 0, 0, hcmLocation)

	old := dailyFilePrefix + "-" + now.In(hcmLocation).AddDate(0, 0, -15).Format(dailyFileLayout) + ".log"
	boundary := dailyFilePrefix + "-" + now.In(hcmLocation).AddDate(0, 0, -14).Format(dailyFileLayout) + ".log"
	recent := dailyFilePrefix + "-" + now.In(hcmLocation).AddDate(0, 0, -2).Format(dailyFileLayout) + ".log"
	for _, name := range []string{old, boundary, recent} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	// Foreign files must never be touched.
	foreign := "payment-gateway.log"
	if err := os.WriteFile(filepath.Join(dir, foreign), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed foreign: %v", err)
	}

	_, err := newDailyFileWriter(dir, fixedNow(t, now))
	if err != nil {
		t.Fatalf("newDailyFileWriter: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, old)); !os.IsNotExist(err) {
		t.Fatalf("%s should be pruned (err=%v)", old, err)
	}
	for _, kept := range []string{boundary, recent, foreign} {
		if _, err := os.Stat(filepath.Join(dir, kept)); err != nil {
			t.Fatalf("%s should be kept: %v", kept, err)
		}
	}
}

func TestNewLogger_WritesJSONToDailyFile(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(LogConfig{Level: "info", Dir: dir})
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	logger.Info("payr32 probe", "k", "v")

	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no daily file written (err=%v entries=%d)", err, len(entries))
	}
	got, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(got), `"msg":"payr32 probe"`) || !strings.Contains(string(got), `"k":"v"`) {
		t.Fatalf("daily file missing JSON payload: %s", got)
	}
}

// TestGinAccessLogReachesDailyFile proves the PAYR-32 mechanism end to end at
// the code level: the GIN logger middleware writing through the same
// MultiWriter(main.go builds) lands the access line in the daily file.
func TestGinAccessLogReachesDailyFile(t *testing.T) {
	dir := t.TempDir()
	w, err := newDailyFileWriter(dir, time.Now)
	if err != nil {
		t.Fatalf("newDailyFileWriter: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.LoggerWithWriter(io.MultiWriter(os.Stdout, w))) // dual output, same as main.go
	router.GET("/api/v1/projects", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError) // simulate the 31/10 incident shape
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("gin access log missing from daily file (err=%v)", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	line := string(got)
	if !strings.Contains(line, "[GIN]") || !strings.Contains(line, "500") || !strings.Contains(line, "/api/v1/projects") {
		t.Fatalf("gin access line incomplete in file: %s", line)
	}
}

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
				Dir:   "",
			},
			wantErr: false,
		},
		{
			name: "info level",
			cfg: LogConfig{
				Level: "info",
				Dir:   "",
			},
			wantErr: false,
		},
		{
			name: "warn level",
			cfg: LogConfig{
				Level: "warn",
				Dir:   "",
			},
			wantErr: false,
		},
		{
			name: "warning level",
			cfg: LogConfig{
				Level: "warning",
				Dir:   "",
			},
			wantErr: false,
		},
		{
			name: "error level",
			cfg: LogConfig{
				Level: "error",
				Dir:   "",
			},
			wantErr: false,
		},
		{
			name: "unknown level defaults to info",
			cfg: LogConfig{
				Level: "unknown",
				Dir:   "",
			},
			wantErr: false,
		},
		{
			name: "with log dir",
			cfg: LogConfig{
				Level: "info",
				Dir:   t.TempDir(),
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
