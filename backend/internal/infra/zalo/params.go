package zalo

// ParamCaps maps a template_id to the per-parameter character cap enforced by
// the Zalo OA dashboard. Sending a value longer than the cap returns Zalo
// error -1121 ("Tham số vượt giới hạn ký tự"); sending fewer params than the
// template declares returns -1122 ("Template thiếu tham số").
//
// Caps count UTF-8 runes (Vietnamese diacritics are one rune each), matching
// the PHP port's use of mb_substr.
//
// Source of truth: the OA console "Tham số" table for each template. The
// password_reset template OTP-ZNS-v1 (id "617976") declares:
//
//	otp_code             string  cap 30
//	user_fullname        string  cap 30
//	otp_valid_in_minutes "date"  cap 20   (rendered as integer-minutes string)
var ParamCaps = map[string]map[string]int{
	"617976": {
		"otp_code":             30,
		"user_fullname":        30,
		"otp_valid_in_minutes": 20,
	},
}

// ClampParams truncates each string value in data to the per-template cap (in
// runes). Values for params not listed in the cap table are passed through
// unchanged. Port of vfic_zns_clamp_params.
//
// This is a defensive measure: callers should already send correctly-sized
// values, but clamping here means a long users.fullname can never produce a
// -1121 from Zalo.
func ClampParams(templateID string, data map[string]string) map[string]string {
	if data == nil {
		return nil
	}
	caps, ok := ParamCaps[templateID]
	if !ok {
		return data
	}
	out := make(map[string]string, len(data))
	for k, v := range data {
		if max, hasCap := caps[k]; hasCap && max > 0 {
			out[k] = clampRunes(v, max)
		} else {
			out[k] = v
		}
	}
	return out
}

// clampRunes truncates s to at most max runes. If s is already short enough it
// is returned as-is (no allocation).
func clampRunes(s string, max int) string {
	if s == "" || max <= 0 {
		if max <= 0 {
			return ""
		}
		return s
	}
	// Fast path: byte-length check first; only count runes if it could exceed.
	if len(s) <= max {
		return s
	}
	count := 0
	for i := range s {
		if count == max {
			return s[:i]
		}
		count++
	}
	return s
}
