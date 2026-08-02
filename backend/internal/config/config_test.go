package config

import (
	"os"
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "returns default when env not set",
			key:          "NONEXISTENT_KEY",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				_ = os.Setenv(tt.key, tt.envValue)
				defer func() { _ = os.Unsetenv(tt.key) }()
			}

			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{name: "valid int", s: "42", want: 42},
		{name: "zero", s: "0", want: 0},
		{name: "negative", s: "-10", want: -10},
		{name: "invalid", s: "invalid", want: 0},
		{name: "empty", s: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInt(tt.s)
			if got != tt.want {
				t.Errorf("parseInt(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int64
	}{
		{name: "valid int64", s: "9223372036854775807", want: 9223372036854775807},
		{name: "zero", s: "0", want: 0},
		{name: "negative", s: "-100", want: -100},
		{name: "invalid", s: "invalid", want: 0},
		{name: "empty", s: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInt64(tt.s)
			if got != tt.want {
				t.Errorf("parseInt64(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want time.Duration
	}{
		{name: "seconds", s: "10s", want: 10 * time.Second},
		{name: "minutes", s: "5m", want: 5 * time.Minute},
		{name: "hours", s: "1h", want: 1 * time.Hour},
		{name: "invalid", s: "invalid", want: 0},
		{name: "empty", s: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDuration(tt.s)
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{name: "true", s: "true", want: true},
		{name: "false", s: "false", want: false},
		{name: "1", s: "1", want: true},
		{name: "0", s: "0", want: false},
		{name: "invalid", s: "invalid", want: false},
		{name: "empty", s: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBool(tt.s)
			if got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseStringSlice(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []string
	}{
		{
			name: "single value",
			s:    "value1",
			want: []string{"value1"},
		},
		{
			name: "multiple values",
			s:    "value1,value2,value3",
			want: []string{"value1", "value2", "value3"},
		},
		{
			name: "values with spaces",
			s:    "value1 , value2 , value3",
			want: []string{"value1", "value2", "value3"},
		},
		{
			name: "empty string",
			s:    "",
			want: []string{},
		},
		{
			name: "empty values filtered",
			s:    "value1,,value2",
			want: []string{"value1", "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStringSlice(tt.s)
			if len(got) != len(tt.want) {
				t.Errorf("parseStringSlice(%q) length = %v, want %v", tt.s, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseStringSlice(%q)[%d] = %v, want %v", tt.s, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseIntSlice(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []int
	}{
		{
			name: "single value",
			s:    "42",
			want: []int{42},
		},
		{
			name: "multiple values",
			s:    "1,2,3,4,5",
			want: []int{1, 2, 3, 4, 5},
		},
		{
			name: "negative values",
			s:    "-1,0,1",
			want: []int{-1, 0, 1},
		},
		{
			name: "empty string",
			s:    "",
			want: []int{},
		},
		{
			name: "invalid values filtered",
			s:    "1,invalid,2",
			want: []int{1, 2},
		},
		{
			name: "values with spaces",
			s:    "1 , 2 , 3",
			want: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIntSlice(tt.s)
			if len(got) != len(tt.want) {
				t.Errorf("parseIntSlice(%q) length = %v, want %v", tt.s, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseIntSlice(%q)[%d] = %v, want %v", tt.s, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "empty JWT secret rejected",
			cfg: &Config{
				App:      AppConfig{Env: "dev"},
				Auth:     AuthConfig{JWTSecret: ""},
				Security: SecurityConfig{HashSecret: "h", HashSalt: "s"},
			},
			wantErr: true,
		},
		{
			name: "empty hash secret rejected",
			cfg: &Config{
				App:      AppConfig{Env: "dev"},
				Auth:     AuthConfig{JWTSecret: "j"},
				Security: SecurityConfig{HashSecret: "", HashSalt: "s"},
			},
			wantErr: true,
		},
		{
			name: "empty hash salt rejected",
			cfg: &Config{
				App:      AppConfig{Env: "dev"},
				Auth:     AuthConfig{JWTSecret: "j"},
				Security: SecurityConfig{HashSecret: "h", HashSalt: ""},
			},
			wantErr: true,
		},
		{
			name: "all non-empty passes in dev",
			cfg: &Config{
				App:      AppConfig{Env: "dev"},
				Auth:     AuthConfig{JWTSecret: "j"},
				Security: SecurityConfig{HashSecret: "h", HashSalt: "s"},
			},
			wantErr: false,
		},
		{
			name: "all non-empty passes in production",
			cfg: &Config{
				App:      AppConfig{Env: "production"},
				Auth:     AuthConfig{JWTSecret: "j"},
				Security: SecurityConfig{HashSecret: "h", HashSalt: "s"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Set environment variables for testing
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("APP_PORT", "9090")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	_ = os.Setenv("HASH_SECRET", "test-hash-secret")
	_ = os.Setenv("HASH_SALT", "test-hash-salt")
	// The test asserts Load() succeeds in a minimal env. Disable Google
	// OAuth so the GOOGLE_CLIENT_ID-must-be-set rule doesn't fire from
	// values leaking in from .env via the test runner; this test isn't
	// about OAuth validation.
	_ = os.Setenv("GOOGLE_OAUTH_ENABLED", "false")
	defer func() {
		_ = os.Unsetenv("APP_ENV")
		_ = os.Unsetenv("APP_PORT")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("HASH_SECRET")
		_ = os.Unsetenv("HASH_SALT")
		_ = os.Unsetenv("GOOGLE_OAUTH_ENABLED")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Env != "test" {
		t.Errorf("Load() App.Env = %v, want test", cfg.App.Env)
	}

	if cfg.App.Port != "9090" {
		t.Errorf("Load() App.Port = %v, want 9090", cfg.App.Port)
	}

	if cfg.Auth.JWTSecret != "test-secret" {
		t.Errorf("Load() Auth.JWTSecret = %v, want test-secret", cfg.Auth.JWTSecret)
	}
}
