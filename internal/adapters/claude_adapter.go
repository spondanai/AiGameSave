package adapters

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/spondanai/AiGameSave/internal/domain"
)

// ClaudeAdapter is the adapter for Claude Code AI CLI.
type ClaudeAdapter struct{}

func NewClaudeAdapter() *ClaudeAdapter {
	return &ClaudeAdapter{}
}

// Detect checks if Claude Code is active by looking for its configuration/history folder.
func (c *ClaudeAdapter) Detect(workingDir string) bool {
	historyPath := filepath.Join(workingDir, ".claude", "history.jsonl")
	_, err := os.Stat(historyPath)
	return err == nil
}

// Extract parses the `.claude/history.jsonl` file.
// It extracts the last 3 turns (role/content pairs) and truncates long messages.
func (c *ClaudeAdapter) Extract(workingDir string) (domain.SessionState, error) {
	historyPath := filepath.Join(workingDir, ".claude", "history.jsonl")

	file, err := os.Open(historyPath)
	if err != nil {
		return domain.SessionState{}, err
	}
	defer file.Close()

	var turns []domain.Turn
	scanner := bufio.NewScanner(file)

	// In MVP, we use a flexible struct to catch common JSON structures
	// used by AI clients if the exact schema isn't strictly known.
	// Claude's history.jsonl might have `role` and `content` (or similar fields).
	type ClaudeLine struct {
		Role    string `json:"role"`
		Type    string `json:"type"`
		Content string `json:"content"`
		Message string `json:"message"`
		Text    string `json:"text"`
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		var cl ClaudeLine

		if err := json.Unmarshal(line, &cl); err == nil {
			role := cl.Role
			if role == "" {
				role = cl.Type
			}
			if role == "" {
				role = "unknown"
			}

			content := cl.Content
			if content == "" {
				content = cl.Message
			}
			if content == "" {
				content = cl.Text
			}

			// Add turn if we successfully got content
			if content != "" {
				// Prevent extremely long tool outputs or logs from eating up context space
				if len(content) > 1500 {
					content = content[:1500] + "\n... [Content truncated for brevity] ..."
				}
				turns = append(turns, domain.Turn{Role: role, Content: strings.TrimSpace(content)})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return domain.SessionState{}, err
	}

	// Keep only the last 6 turns (3 pairs roughly)
	maxTurns := 6
	if len(turns) > maxTurns {
		turns = turns[len(turns)-maxTurns:]
	}

	return domain.SessionState{
		RecentTurns: turns,
	}, nil
}

func (c *ClaudeAdapter) Name() string {
	return "Claude Code"
}
