package adapters

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spondanai/aigamesave/internal/domain"
)

// AiderAdapter is the adapter for Aider AI CLI.
type AiderAdapter struct{}

func NewAiderAdapter() *AiderAdapter {
	return &AiderAdapter{}
}

// Detect checks if Aider is active by looking for its history file.
func (a *AiderAdapter) Detect(workingDir string) bool {
	historyPath := filepath.Join(workingDir, ".aider.chat.history.md")
	_, err := os.Stat(historyPath)
	return err == nil
}

func (a *AiderAdapter) LastActive(workingDir string) (time.Time, error) {
	historyPath := filepath.Join(workingDir, ".aider.chat.history.md")
	info, err := os.Stat(historyPath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// Extract parses the `.aider.chat.history.md` file.
// It extracts the last 3 turns (USER/ASSISTANT pairs) and truncates long code blocks.
func (a *AiderAdapter) Extract(workingDir string) (domain.SessionState, error) {
	historyPath := filepath.Join(workingDir, ".aider.chat.history.md")

	file, err := os.Open(historyPath)
	if err != nil {
		return domain.SessionState{}, err
	}
	defer file.Close()

	var turns []domain.Turn
	var currentRole string
	var currentContent strings.Builder

	scanner := bufio.NewScanner(file)
	inCodeBlock := false
	codeBlockLines := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Detect Aider turn headers (typically starts with # USER or # ASSISTANT)
		if strings.HasPrefix(line, "# USER") || strings.HasPrefix(line, "#### /") || strings.HasPrefix(line, "> USER") {
			if currentRole != "" {
				turns = append(turns, domain.Turn{Role: currentRole, Content: strings.TrimSpace(currentContent.String())})
				currentContent.Reset()
			}
			currentRole = "user"
			continue
		} else if strings.HasPrefix(line, "# ASSISTANT") || strings.HasPrefix(line, "> ASSISTANT") {
			if currentRole != "" {
				turns = append(turns, domain.Turn{Role: currentRole, Content: strings.TrimSpace(currentContent.String())})
				currentContent.Reset()
			}
			currentRole = "assistant"
			continue
		}

		// Basic truncation for long code blocks to save context
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				codeBlockLines = 0
			}
		}

		if inCodeBlock {
			codeBlockLines++
			if codeBlockLines > 50 {
				if codeBlockLines == 51 {
					currentContent.WriteString("\n... [Code block truncated for brevity] ...\n")
				}
				continue // Skip appending this line
			}
		}

		// Append line to current turn
		if currentRole != "" {
			currentContent.WriteString(line + "\n")
		}
	}

	// Append the last turn
	if currentRole != "" && currentContent.Len() > 0 {
		turns = append(turns, domain.Turn{Role: currentRole, Content: strings.TrimSpace(currentContent.String())})
	}

	if err := scanner.Err(); err != nil {
		return domain.SessionState{}, err
	}

	var rawBytes int64
	if info, err := os.Stat(historyPath); err == nil {
		rawBytes = info.Size()
	}

	return domain.SessionState{
		RecentTurns: selectContext(turns),
		RawBytes:    rawBytes,
	}, nil
}

func (a *AiderAdapter) Name() string {
	return "Aider"
}
