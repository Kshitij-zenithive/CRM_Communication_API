// internal/util/util.go (No changes needed)
package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// GenerateRandomString generates a random string of the specified length
func GenerateRandomString(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		// Fall back to a hardcoded string if random generation fails
		return "fallback-random-string-for-testing-purposes-only"
	}

	return base64.URLEncoding.EncodeToString(b)[:length]
}

// ExtractMentions extracts @mentions from text
func ExtractMentions(text string) []string {
	re := regexp.MustCompile(`@(\w+)`)
	matches := re.FindAllStringSubmatch(text, -1)

	var mentions []string
	mentionMap := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 && !mentionMap[match[1]] {
			mentions = append(mentions, match[1])
			mentionMap[match[1]] = true
		}
	}

	return mentions
}

// FormatEmailAddress formats a name and email address
func FormatEmailAddress(name, email string) string {
	return fmt.Sprintf("%s <%s>", name, email)
}

// SanitizeHTML sanitizes HTML content
func SanitizeHTML(html string) string {
	// This is a very basic implementation
	// In a production environment, use a proper HTML sanitizer library

	// Remove script tags
	scriptPattern := regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`)
	html = scriptPattern.ReplaceAllString(html, "")

	// Remove on* attributes
	onEventPattern := regexp.MustCompile(`\s+on\w+="[^"]*"`)
	html = onEventPattern.ReplaceAllString(html, "")

	return html
}

// TruncateString truncates a string to the specified length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}

// NormalizeEmail normalizes an email address (lowercase, trim)
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ParseDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 -0700", // Common format
		"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
	}

	var parsedTime time.Time
	var err error

	for _, format := range formats {
		parsedTime, err = time.Parse(format, dateStr)
		if err == nil {
			return parsedTime, nil
		}
	}
	return parsedTime, fmt.Errorf("failed to parse date string: %s, last err: %w", dateStr, err)

}

func DecodeBase64String(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded) // Use standard encoding
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(encoded) // Try URL encoding
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 string: %w", err)
		}
	}
	return string(data), nil
}
