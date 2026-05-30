package ua

import (
	"strings"

	useragent "github.com/mssola/useragent"
)

// Parse extracts a human-readable browser and platform from a raw User-Agent string.
// Returns ("Unknown", "Unknown") for empty input.
func Parse(rawUA string) (browser, platform string) {
	if rawUA == "" {
		return "Unknown", "Unknown"
	}

	parsed := useragent.New(rawUA)
	browserName, browserVer := parsed.Browser()
	browser = formatBrowser(browserName, browserVer)
	platform = formatPlatform(parsed.OSInfo())

	return browser, platform
}

// formatBrowser returns "Name MajorVersion" (e.g., "Chrome 147").
func formatBrowser(name, version string) string {
	if name == "" {
		return "Unknown"
	}
	major := majorVersion(version)
	if major == "" {
		return name
	}
	return name + " " + major
}

// majorVersion extracts the major version segment (before the first dot).
func majorVersion(v string) string {
	if v == "" {
		return ""
	}
	major, _, _ := strings.Cut(v, ".")
	return major
}

// formatPlatform normalizes OS info into "Name Version" (e.g., "macOS 15.7").
func formatPlatform(info useragent.OSInfo) string {
	name := info.Name
	version := info.Version

	// Normalize common names
	switch name {
	case "Mac OS":
		name = "macOS"
	case "Mac OS X":
		name = "macOS"
	case "Intel Mac OS":
		name = "macOS"
	case "iPhone OS":
		name = "iOS"
	case "iPad OS":
		name = "iPadOS"
	}

	if version == "" {
		if name == "" {
			return "Unknown"
		}
		return name
	}
	return name + " " + version
}
