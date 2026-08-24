package utils

import "strings"

// ParseUserAgent does a best-effort, dependency-free classification of a
// User-Agent string into (device type, OS, browser). It intentionally
// avoids extracting anything more specific/identifying than that.
func ParseUserAgent(ua string) (deviceType, osName, browserName string) {
	lower := strings.ToLower(ua)

	switch {
	case strings.Contains(lower, "ipad") || strings.Contains(lower, "tablet"):
		deviceType = "tablet"
	case strings.Contains(lower, "mobi") || strings.Contains(lower, "iphone") || strings.Contains(lower, "android"):
		deviceType = "mobile"
	case ua == "":
		deviceType = "unknown"
	default:
		deviceType = "desktop"
	}

	switch {
	case strings.Contains(lower, "windows"):
		osName = "Windows"
	case strings.Contains(lower, "mac os") || strings.Contains(lower, "macos"):
		osName = "macOS"
	case strings.Contains(lower, "android"):
		osName = "Android"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad") || strings.Contains(lower, "ios"):
		osName = "iOS"
	case strings.Contains(lower, "linux"):
		osName = "Linux"
	default:
		osName = "Other"
	}

	switch {
	case strings.Contains(lower, "edg/"):
		browserName = "Edge"
	case strings.Contains(lower, "chrome/") && !strings.Contains(lower, "chromium"):
		browserName = "Chrome"
	case strings.Contains(lower, "firefox/"):
		browserName = "Firefox"
	case strings.Contains(lower, "safari/") && !strings.Contains(lower, "chrome/"):
		browserName = "Safari"
	default:
		browserName = "Other"
	}

	return deviceType, osName, browserName
}
