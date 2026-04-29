package redaction

import (
	"regexp"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`sk-ant-[A-Za-z0-9-_]{32,}`),                                      // Anthropic
	regexp.MustCompile(`sk-[A-Za-z0-9-_]{32,}`),                                          // OpenAI-like
	regexp.MustCompile(`xox[baprs]-[0-9]{12,}-[0-9]{12,}-[a-zA-Z0-9]{24}`),              // Slack
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`),                                     // GitHub tokens
	regexp.MustCompile(`AKIA[A-Z0-9]{16}`),                                                // AWS Access Key ID
	regexp.MustCompile(`(?i)aws_secret_access_key\s*=\s*[A-Za-z0-9/+=]{40}`),            // AWS Secret Access Key
	regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),                                         // Google API key
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-._~+/]{20,}`),                          // Generic Bearer token
}

// MaskSecrets replaces known secret patterns with a placeholder.
func MaskSecrets(content string) string {
	redacted := content
	for _, pattern := range secretPatterns {
		redacted = pattern.ReplaceAllString(redacted, "[REDACTED_API_KEY]")
	}
	return redacted
}
