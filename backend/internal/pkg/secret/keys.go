package secret

// ZaloCredentialsKey is the settings row holding the Zalo OA app secret and the
// OAuth access/refresh tokens. zaloconnect.KeyCredentials is the runtime
// authority for this name; the literal is repeated here because a low-level
// crypto package must not depend on an application service. The two are pinned
// together by TestProtectedKeysMatchZaloconnectCredentials.
const ZaloCredentialsKey = "zalo.credentials"

// MimoAPIKey is the settings row holding the Xiaomi MiMo LLM API key. The
// runtime authority for this name is config.SettingKeyMimoAPIKey; the literal
// is repeated here for the same layering reason as above. Registration in
// ProtectedKeys is load-bearing: settings GETs are Authenticate-only and
// partner/employee hold GET grants on /settings/*, so an unsealed value would
// be readable by every authenticated role. Masking in API responses is
// allowlist-driven too (dto.NewSettingResponse → IsProtectedKey).
const MimoAPIKey = "mimo.api_key"

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
	MimoAPIKey,
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
