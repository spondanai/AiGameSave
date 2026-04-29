package adapters

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spondanai/aigamesave/internal/domain"
)

// CopilotAdapter reads GitHub Copilot Chat sessions stored by VS Code.
// Sessions live in ~/Library/Application Support/Code/User/workspaceStorage/<hash>/chatSessions/*.json
type CopilotAdapter struct{}

func NewCopilotAdapter() *CopilotAdapter {
	return &CopilotAdapter{}
}

func vscodeWorkspaceStorageRoot() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "workspaceStorage")
}

// findWorkspaceStorageDir scans workspaceStorage for a folder whose workspace.json
// folder field matches workingDir or any of its parents.
func findWorkspaceStorageDir(workingDir string) string {
	storageRoot := vscodeWorkspaceStorageRoot()
	if storageRoot == "" {
		return ""
	}

	entries, err := os.ReadDir(storageRoot)
	if err != nil {
		return ""
	}

	// Build candidate set: workingDir and all parents up to home.
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

	var best struct {
		path    string
		modTime time.Time
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		wsFile := filepath.Join(storageRoot, e.Name(), "workspace.json")
		folder, err := readWorkspaceFolder(wsFile)
		if err != nil {
			continue
		}
		if _, ok := candidates[folder]; !ok {
			continue
		}
		sessionsDir := filepath.Join(storageRoot, e.Name(), "chatSessions")
		info, err := os.Stat(sessionsDir)
		if err != nil || !info.IsDir() {
			continue
		}
		if info.ModTime().After(best.modTime) {
			best.path = filepath.Join(storageRoot, e.Name())
			best.modTime = info.ModTime()
		}
	}
	return best.path
}

func readWorkspaceFolder(wsFile string) (string, error) {
	data, err := os.ReadFile(wsFile)
	if err != nil {
		return "", err
	}
	var ws struct {
		Folder string `json:"folder"`
	}
	if err := json.Unmarshal(data, &ws); err != nil {
		return "", err
	}
	// folder is a file URI: "file:///path/with%20spaces"
	raw := strings.TrimPrefix(ws.Folder, "file://")
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return filepath.Clean(raw), nil
	}
	return filepath.Clean(decoded), nil
}

type copilotSession struct {
	SessionID       string            `json:"sessionId"`
	LastMessageDate int64             `json:"lastMessageDate"`
	Requests        []copilotRequest  `json:"requests"`
}

type copilotRequest struct {
	Message   copilotMessage   `json:"message"`
	Response  []copilotResponse `json:"response"`
	Timestamp int64            `json:"timestamp"`
}

type copilotMessage struct {
	Text string `json:"text"`
}

type copilotResponse struct {
	Value string `json:"value"`
}

func (c *CopilotAdapter) latestSession(workingDir string) (*copilotSession, error) {
	wsDir := findWorkspaceStorageDir(workingDir)
	if wsDir == "" {
		return nil, fmt.Errorf("no VS Code workspace found for %s", workingDir)
	}

	sessionsDir := filepath.Join(wsDir, "chatSessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read chatSessions: %w", err)
	}

	type candidate struct {
		path            string
		lastMessageDate int64
	}
	var candidates []candidate

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(sessionsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var s copilotSession
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		if len(s.Requests) == 0 {
			continue
		}
		candidates = append(candidates, candidate{path: path, lastMessageDate: s.LastMessageDate})
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no Copilot Chat sessions with messages found for %s", workingDir)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].lastMessageDate > candidates[j].lastMessageDate
	})

	data, err := os.ReadFile(candidates[0].path)
	if err != nil {
		return nil, err
	}
	var s copilotSession
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *CopilotAdapter) Detect(workingDir string) bool {
	_, err := c.latestSession(workingDir)
	return err == nil
}

func (c *CopilotAdapter) LastActive(workingDir string) (time.Time, error) {
	s, err := c.latestSession(workingDir)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(s.LastMessageDate), nil
}

func (c *CopilotAdapter) Extract(workingDir string) (domain.SessionState, error) {
	s, err := c.latestSession(workingDir)
	if err != nil {
		return domain.SessionState{}, err
	}

	var turns []domain.Turn
	for _, req := range s.Requests {
		userText := strings.TrimSpace(req.Message.Text)
		if userText != "" {
			if len(userText) > 1500 {
				userText = userText[:1500] + "\n... [Content truncated for brevity] ..."
			}
			turns = append(turns, domain.Turn{Role: "user", Content: userText})
		}

		var parts []string
		for _, r := range req.Response {
			if v := strings.TrimSpace(r.Value); v != "" {
				parts = append(parts, v)
			}
		}
		if assistantText := strings.TrimSpace(strings.Join(parts, "")); assistantText != "" {
			if len(assistantText) > 1500 {
				assistantText = assistantText[:1500] + "\n... [Content truncated for brevity] ..."
			}
			turns = append(turns, domain.Turn{Role: "assistant", Content: assistantText})
		}
	}

	maxTurns := 6
	if len(turns) > maxTurns {
		turns = turns[len(turns)-maxTurns:]
	}

	return domain.SessionState{RecentTurns: turns}, nil
}

func (c *CopilotAdapter) Name() string {
	return "GitHub Copilot"
}
