package ua

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		browser  string
		platform string
	}{
		{
			name:     "empty string",
			ua:       "",
			browser:  "Unknown",
			platform: "Unknown",
		},
		{
			name:     "Chrome on macOS",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36",
			browser:  "Chrome 147",
			platform: "macOS 10.15.7",
		},
		{
			name:     "Safari on macOS",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_3_2) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Safari/605.1.15",
			browser:  "Safari 18",
			platform: "macOS 15.3.2",
		},
		{
			name:     "Firefox on Windows",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:137.0) Gecko/20100101 Firefox/137.0",
			browser:  "Firefox 137",
			platform: "Windows 10",
		},
		{
			name:     "Chrome on Windows",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
			browser:  "Chrome 136",
			platform: "Windows 10",
		},
		{
			name:     "Safari on iPhone",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 18_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.4 Mobile/15E148 Safari/604.1",
			browser:  "Safari 18",
			platform: "iOS 18.4",
		},
		{
			name:     "Chrome on Android",
			ua:       "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.7049.100 Mobile Safari/537.36",
			browser:  "Chrome 135",
			platform: "Android 15",
		},
		{
			name:     "Edge on Windows",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36 Edg/136.0.0.0",
			browser:  "Edge 136",
			platform: "Windows 10",
		},
		{
			name:     "Opera on macOS",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36 OPR/121.0.0.0",
			browser:  "Opera 121",
			platform: "macOS 10.15.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			browser, platform := Parse(tt.ua)
			if browser != tt.browser {
				t.Errorf("browser = %q, want %q", browser, tt.browser)
			}
			if platform != tt.platform {
				t.Errorf("platform = %q, want %q", platform, tt.platform)
			}
		})
	}
}
