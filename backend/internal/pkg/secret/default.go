package secret

import (
	"os"
	"sync"

	"api-server/internal/infra/observability"
)

var (
	defaultMu       sync.Mutex
	defaultCipher   *Cipher
	defaultResolved bool
)

// logWarn and logInfo are seams so tests can assert the degraded-mode warning
// without initializing the application file logger.
var (
	logWarn = func(msg string, args ...any) { observability.GetLogger().Warn(msg, args...) }
	logInfo = func(msg string, args ...any) { observability.GetLogger().Info(msg, args...) }
)

// Default returns the process-wide cipher, resolving it from SETTINGS_SECRET_KEY
// on first use. It returns nil when no usable key is configured, and never
// panics or errors: an operator who forgot the key still gets a working API.
//
// The value is resolved exactly once per process, so the degraded-mode warning
// is logged once no matter how many callers ask.
func Default() *Cipher {
	return resolveDefault()
}

// LoadFromEnv resolves the cipher from SETTINGS_SECRET_KEY, installs it as the
// process-wide default and returns it (nil when encryption stays disabled).
//
// Bootstrap may call it once at startup so the warning lands with the rest of
// the boot log; otherwise Default resolves lazily on first use. Calling it more
// than once is harmless — the first call wins.
func LoadFromEnv() *Cipher {
	return resolveDefault()
}

// SetDefault installs an explicit cipher — used by tests, and by wiring that
// builds the cipher from application config rather than reading the environment
// directly. Passing nil disables encryption for the process.
func SetDefault(c *Cipher) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultCipher = c
	defaultResolved = true
}

func resolveDefault() *Cipher {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultResolved {
		return defaultCipher
	}
	defaultResolved = true

	encoded := os.Getenv(EnvKey)
	if encoded == "" {
		logWarn(
			"settings secret encryption disabled: "+EnvKey+" is not set — protected integration settings are stored in plaintext",
			"protected_keys", ProtectedKeys,
		)
		return nil
	}

	c, err := NewCipherFromBase64(encoded)
	if err != nil {
		// A malformed key must not fail startup either: behave as if no key were
		// configured, but say so loudly, since any value sealed under the real
		// key is now unreadable.
		logWarn(
			"settings secret encryption disabled: "+EnvKey+" is not a base64-encoded 32-byte key",
			"error", err,
			"protected_keys", ProtectedKeys,
		)
		return nil
	}

	defaultCipher = c
	logInfo(
		"settings secret encryption enabled",
		"protected_keys", ProtectedKeys,
	)
	return c
}

// resetDefaultForTest clears the process-wide resolution so a test can drive
// Default/LoadFromEnv with a different environment.
func resetDefaultForTest() {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultCipher = nil
	defaultResolved = false
}
