package secret

// ZaloCredentialsKey is the settings row holding the Zalo OA app secret and the
// OAuth access/refresh tokens. zaloconnect.KeyCredentials is the runtime
// authority for this name; the literal is repeated here because a low-level
// crypto package must not depend on an application service. The two are pinned
// together by TestProtectedKeysMatchZaloconnectCredentials.
const ZaloCredentialsKey = "zalo.credentials"

// ProtectedKeys is the complete set of settings keys whose values are sealed at
// rest. Deliberately an allow-list, not a heuristic: sealing a value that an
// operator needs to read by hand (or a key nobody ever rotates) buys nothing,
// and a wrong guess breaks the feature the setting drives.
//
// Adding a key here is safe — plaintext values stay readable and the startup
// backfill converts them — but any code that compares stored values literally
// (settings CompareAndSwapValue) must go through the repository, never raw SQL.
var ProtectedKeys = []string{
	ZaloCredentialsKey,
}

// IsProtectedKey reports whether a settings key holds a sealed value.
func IsProtectedKey(key string) bool {
	for _, protected := range ProtectedKeys {
		if key == protected {
			return true
		}
	}
	return false
}
