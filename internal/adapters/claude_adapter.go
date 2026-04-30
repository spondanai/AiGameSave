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

type ClaudeAdapter struct {
	// projectDirOverride bypasses findClaudeProjectDir when set; used in tests.
	projectDirOverride string
}

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

func (c *ClaudeAdapter) LastActive(workingDir string) (time.Time, error) {
	projectDir := findClaudeProjectDir(workingDir)
	if projectDir == "" {
		return time.Time{}, fmt.Errorf("no Claude Code session found for %s", workingDir)
	}
	path, err := latestJSONL(projectDir)
	if err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// latestJSONL returns the path to the most recently modified .jsonl file in dir.
func latestJSONL(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("cannot read dir %s: %w", dir, err)
	}
	type fi struct {
		path    string
		modTime time.Time
	}
	var files []fi
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fi{filepath.Join(dir, e.Name()), info.ModTime()})
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no session files found in %s", dir)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modTime.After(files[j].modTime) })
	return files[0].path, nil
}

// Extract parses the latest Claude Code session file.
func (c *ClaudeAdapter) Extract(workingDir string) (domain.SessionState, error) {
	projectDir := c.projectDirOverride
	if projectDir == "" {
		projectDir = findClaudeProjectDir(workingDir)
		if projectDir == "" {
			return domain.SessionState{}, fmt.Errorf("no Claude Code session found for %s", workingDir)
		}
	}
	return c.extractFromPath(projectDir)
}

// extractFromPath finds the newest session file inside projectDir and parses it.
func (c *ClaudeAdapter) extractFromPath(projectDir string) (domain.SessionState, error) {
	sessionPath, err := latestJSONL(projectDir)
	if err != nil {
		return domain.SessionState{}, err
	}

	file, err := os.Open(sessionPath)
	if err != nil {
		return domain.SessionState{}, err
	}
	defer file.Close()

	type ContentItem struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		Name  string          `json:"name"`  // tool_use: tool name
		Input json.RawMessage `json:"input"` // tool_use: arguments
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
	type TodoItem struct {
		Content  string `json:"content"`
		Status   string `json:"status"` // pending | in_progress | completed
		Priority string `json:"priority"`
	}
	type TodoWriteInput struct {
		Todos []TodoItem `json:"todos"`
	}

	var turns []domain.Turn
	var latestTodos []TodoItem // overwritten on every TodoWrite call
	reader := bufio.NewReader(file)

	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			break
		}

		var fullLine []byte
		fullLine = append(fullLine, line...)

		for isPrefix {
			line, isPrefix, err = reader.ReadLine()
			if err != nil {
				break
			}
			// Cap line size to ~5MB to prevent OOM
			if len(fullLine) < 5*1024*1024 {
				fullLine = append(fullLine, line...)
			}
		}

		if len(fullLine) >= 5*1024*1024 {
			continue // Skip overly large lines
		}

		var cl ClaudeLine
		if err := json.Unmarshal(fullLine, &cl); err != nil {
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
					switch item.Type {
					case "text":
						if item.Text != "" {
							parts = append(parts, item.Text)
						}
					case "tool_use":
						if item.Name == "TodoWrite" {
							var input TodoWriteInput
							if err := json.Unmarshal(item.Input, &input); err == nil {
								latestTodos = input.Todos
							}
						}
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

	// Collect pending/in_progress tasks from the last TodoWrite call.
	var pending []string
	for _, t := range latestTodos {
		if t.Status == "pending" || t.Status == "in_progress" {
			pending = append(pending, t.Content)
		}
	}

	var rawBytes int64
	if info, err := os.Stat(sessionPath); err == nil {
		rawBytes = info.Size()
	}

	return domain.SessionState{
		RecentTurns:  selectContext(turns),
		PendingTasks: pending,
		RawBytes:     rawBytes,
	}, nil
}

func (c *ClaudeAdapter) Name() string {
	return "Claude Code"
}
