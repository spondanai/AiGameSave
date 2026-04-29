package adapters

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spondanai/aigamesave/internal/domain"
)

type ClaudeAdapter struct{}

func NewClaudeAdapter() *ClaudeAdapter {
	return &ClaudeAdapter{}
}

// encodeClaudePath mirrors Claude Code's directory encoding:
// both "/" and "." are replaced with "-".
func encodeClaudePath(dir string) string {
	r := strings.NewReplacer("/", "-", ".", "-")
	return r.Replace(dir)
}

// findClaudeProjectDir walks up from workingDir toward home, returning the first
// ~/.claude/projects/<encoded-dir> that contains at least one .jsonl session file.
// This handles the common case where Claude Code is launched from a parent directory
// (e.g., home) while ags is run from a subdirectory.
func findClaudeProjectDir(workingDir string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	projectsRoot := filepath.Join(homeDir, ".claude", "projects")

	dir := workingDir
	for {
		encoded := encodeClaudePath(dir)
		candidate := filepath.Join(projectsRoot, encoded)

		entries, err := os.ReadDir(candidate)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
					return candidate
				}
			}
		}

		// Stop at home directory — no point going higher.
		if dir == homeDir || dir == "/" || dir == "." {
			break
		}
		dir = filepath.Dir(dir)
	}
	return ""
}

// Detect checks if Claude Code has a session for the working directory or any parent.
func (c *ClaudeAdapter) Detect(workingDir string) bool {
	return findClaudeProjectDir(workingDir) != ""
}

// getLatestSession returns the path to the most recently modified session file.
func (c *ClaudeAdapter) getLatestSession(workingDir string) (string, error) {
	projectDir := findClaudeProjectDir(workingDir)

	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "", fmt.Errorf("cannot read claude project dir: %w", err)
	}

	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var files []fileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{
			path:    filepath.Join(projectDir, e.Name()),
			modTime: info.ModTime(),
		})
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no session files found in %s", projectDir)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	return files[0].path, nil
}

// Extract parses the latest Claude Code session file.
// User messages have content as a plain string.
// Assistant messages have content as an array of typed blocks; we extract "text" blocks.
func (c *ClaudeAdapter) Extract(workingDir string) (domain.SessionState, error) {
	sessionPath, err := c.getLatestSession(workingDir)
	if err != nil {
		return domain.SessionState{}, err
	}

	file, err := os.Open(sessionPath)
	if err != nil {
		return domain.SessionState{}, err
	}
	defer file.Close()

	type ContentItem struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	type ClaudeLine struct {
		Type    string  `json:"type"`
		IsMeta  bool    `json:"isMeta"`
		Message Message `json:"message"`
	}

	var turns []domain.Turn
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var cl ClaudeLine
		if err := json.Unmarshal(scanner.Bytes(), &cl); err != nil {
			continue
		}

		if cl.IsMeta || (cl.Type != "user" && cl.Type != "assistant") {
			continue
		}

		var content string
		// User content is a plain string; assistant content is []ContentItem.
		if err := json.Unmarshal(cl.Message.Content, &content); err != nil {
			var items []ContentItem
			if err := json.Unmarshal(cl.Message.Content, &items); err == nil {
				var parts []string
				for _, item := range items {
					if item.Type == "text" && item.Text != "" {
						parts = append(parts, item.Text)
					}
				}
				content = strings.Join(parts, "\n")
			}
		}

		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}

		if len(content) > 1500 {
			content = content[:1500] + "\n... [Content truncated for brevity] ..."
		}

		turns = append(turns, domain.Turn{Role: cl.Type, Content: content})
	}

	if err := scanner.Err(); err != nil {
		return domain.SessionState{}, err
	}

	maxTurns := 6
	if len(turns) > maxTurns {
		turns = turns[len(turns)-maxTurns:]
	}

	return domain.SessionState{RecentTurns: turns}, nil
}

func (c *ClaudeAdapter) Name() string {
	return "Claude Code"
}
