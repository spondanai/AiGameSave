package adapters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spondanai/aigamesave/internal/domain"
)

// ClineAdapter reads Cline (VS Code extension) task conversations.
// Cline stores tasks in:
//
//	macOS:   ~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/tasks/<ts>/
//	Linux:   ~/.config/Code/User/globalStorage/saoudrizwan.claude-dev/tasks/<ts>/
//	Windows: %APPDATA%\Code\User\globalStorage\saoudrizwan.claude-dev\tasks\<ts>\
//
// Each task directory contains api_conversation_history.json (a JSON array of
// {role, content} messages). The cwd is embedded in the first user message's
// environment_details block as "Current Working Directory (/path)".
type ClineAdapter struct {
	cachedTask string
}

func NewClineAdapter() *ClineAdapter {
	return &ClineAdapter{}
}

func clineGlobalStorageRoot() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	const extID = "saoudrizwan.claude-dev"
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Code", "User", "globalStorage", extID, "tasks")
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "globalStorage", extID, "tasks")
	default:
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(homeDir, ".config")
		}
		return filepath.Join(configDir, "Code", "User", "globalStorage", extID, "tasks")
	}
}

// clineMessage mirrors one entry in api_conversation_history.json.
type clineMessage struct {
	Role    string            `json:"role"`
	Content json.RawMessage   `json:"content"`
}

// clineContentItem is one block inside a content array.
type clineContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// cwdPattern extracts the path from Cline's environment_details header, e.g.:
//
//	"# Current Working Directory (/some/path) Files"
var cwdPattern = regexp.MustCompile(`Current Working Directory \(([^)]+)\)`)

// extractClineCWD reads the first user message from the conversation file and
// returns the working directory Cline was running in.
func extractClineCWD(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var msgs []clineMessage
	if err := json.Unmarshal(data, &msgs); err != nil {
		return ""
	}
	for _, m := range msgs {
		if m.Role != "user" {
			continue
		}
		text := clineMessageText(m.Content)
		if m := cwdPattern.FindStringSubmatch(text); len(m) == 2 {
			return filepath.Clean(m[1])
		}
	}
	return ""
}

func clineMessageText(raw json.RawMessage) string {
	// content can be a plain string or an array of typed blocks.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var items []clineContentItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return ""
	}
	var parts []string
	for _, item := range items {
		if item.Text != "" {
			parts = append(parts, item.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// taskTimestamp extracts the numeric timestamp from the task directory name.
// Cline names task dirs after their creation time in milliseconds.
func taskTimestamp(name string) int64 {
	ts, err := strconv.ParseInt(name, 10, 64)
	if err != nil {
		return 0
	}
	return ts
}

// clineFindTaskIn is the testable core of findTask — it searches inside tasksRoot
// instead of the default global storage location.
func clineFindTaskIn(tasksRoot, workingDir string) (string, error) {
	entries, err := os.ReadDir(tasksRoot)
	if err != nil {
		return "", fmt.Errorf("no Cline tasks dir: %w", err)
	}

	homeDir, _ := os.UserHomeDir()
	candidates := make(map[string]struct{})
	dir := filepath.Clean(workingDir)
	for {
		candidates[dir] = struct{}{}
		if dir == homeDir || dir == "/" || dir == "." {
			break
		}
		dir = filepath.Dir(dir)
	}

	type taskEntry struct {
		path string
		ts   int64
	}
	var matched []taskEntry

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		histPath := filepath.Join(tasksRoot, e.Name(), "api_conversation_history.json")
		cwd := extractClineCWD(histPath)
		if cwd == "" {
			continue
		}
		if _, ok := candidates[cwd]; !ok {
			continue
		}
		matched = append(matched, taskEntry{path: histPath, ts: taskTimestamp(e.Name())})
	}

	if len(matched) == 0 {
		return "", fmt.Errorf("no Cline task found for %s", workingDir)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].ts > matched[j].ts
	})

	return matched[0].path, nil
}

func (c *ClineAdapter) findTask(workingDir string) (string, error) {
	if c.cachedTask != "" {
		return c.cachedTask, nil
	}

	tasksRoot := clineGlobalStorageRoot()
	if tasksRoot == "" {
		return "", fmt.Errorf("cannot locate Cline global storage")
	}

	path, err := clineFindTaskIn(tasksRoot, workingDir)
	if err != nil {
		return "", err
	}
	c.cachedTask = path
	return path, nil
}

func (c *ClineAdapter) Detect(workingDir string) bool {
	_, err := c.findTask(workingDir)
	return err == nil
}

func (c *ClineAdapter) LastActive(workingDir string) (time.Time, error) {
	taskPath, err := c.findTask(workingDir)
	if err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(taskPath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (c *ClineAdapter) Extract(workingDir string) (domain.SessionState, error) {
	taskPath, err := c.findTask(workingDir)
	if err != nil {
		return domain.SessionState{}, err
	}

	data, err := os.ReadFile(taskPath)
	if err != nil {
		return domain.SessionState{}, fmt.Errorf("cannot read Cline task: %w", err)
	}

	var msgs []clineMessage
	if err := json.Unmarshal(data, &msgs); err != nil {
		return domain.SessionState{}, fmt.Errorf("cannot parse Cline task: %w", err)
	}

	var turns []domain.Turn
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		content := strings.TrimSpace(clineMessageText(m.Content))
		if content == "" {
			continue
		}
		if len(content) > 1500 {
			content = content[:1500] + "\n... [Content truncated for brevity] ..."
		}
		turns = append(turns, domain.Turn{Role: m.Role, Content: content})
	}

	var rawBytes int64
	if info, err := os.Stat(taskPath); err == nil {
		rawBytes = info.Size()
	}

	return domain.SessionState{RecentTurns: selectContext(turns), RawBytes: rawBytes}, nil
}

func (c *ClineAdapter) Name() string {
	return "Cline"
}
