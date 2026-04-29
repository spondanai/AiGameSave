package redaction

import (
	"regexp"
)

// Known patterns for API keys (e.g., OpenAI, Anthropic, generic Bearer tokens).
// This is a naive implementation for MVP.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`sk-[A-Za-z0-9-_]{32,}`),               // Common OpenAI-like
	regexp.MustCompile(`xox[baprs]-[0-9]{12,}-[0-9]{12,}-[a-zA-Z0-9]{24}`), // Slack
	// Add more patterns as needed
}

// MaskSecrets replaces found secrets with a placeholder.
func MaskSecrets(content string) string {
	redacted := content
	for _, pattern := range secretPatterns {
		redacted = pattern.ReplaceAllString(redacted, "[REDACTED_API_KEY]")
	}
	return redacted
}
