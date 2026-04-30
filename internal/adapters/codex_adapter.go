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

const codexMaxLineBytes = 5 * 1024 * 1024

type CodexAdapter struct {
	cachedSession string
}

func NewCodexAdapter() *CodexAdapter {
	return &CodexAdapter{}
}

func codexHomeDir() string {
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return dir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".codex")
}

func codexCandidateDirs(workingDir string) map[string]struct{} {
	homeDir, _ := os.UserHomeDir()
	dirs := make(map[string]struct{})

	dir := filepath.Clean(workingDir)
	for {
		parent := filepath.Dir(dir)
		if dir == homeDir || dir == "/" || dir == "." || parent == dir {
			break
		}
		dirs[dir] = struct{}{}
		dir = parent
	}
	return dirs
}

type codexSessionMeta struct {
	Type    string `json:"type"`
	Payload struct {
		CWD string `json:"cwd"`
	} `json:"payload"`
}

func codexSessionCWD(path string) string {
	line, err := readFirstLine(path, codexMaxLineBytes)
	if err != nil {
		return ""
	}

	var meta codexSessionMeta
	if err := json.Unmarshal(line, &meta); err != nil {
		return ""
	}
	if meta.Type != "session_meta" {
		return ""
	}
	return filepath.Clean(meta.Payload.CWD)
}

func readFirstLine(path string, maxBytes int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	line, _, err := readLimitedLine(reader, maxBytes)
	return line, err
}

func readLimitedLine(reader *bufio.Reader, maxBytes int) ([]byte, bool, error) {
	var fullLine []byte
	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			return nil, false, err
		}
		if len(fullLine) < maxBytes {
			remaining := maxBytes - len(fullLine)
			if len(line) > remaining {
				fullLine = append(fullLine, line[:remaining]...)
			} else {
				fullLine = append(fullLine, line...)
			}
		}
		if !isPrefix {
			return fullLine, len(fullLine) >= maxBytes, nil
		}
	}
}

func (c *CodexAdapter) findSession(workingDir string) (string, error) {
	if c.cachedSession != "" {
		return c.cachedSession, nil
	}

	codexHome := codexHomeDir()
	if codexHome == "" {
		return "", fmt.Errorf("cannot locate Codex home")
	}

	allowed := codexCandidateDirs(workingDir)
	sessionsRoot := filepath.Join(codexHome, "sessions")

	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var matched []fileInfo

	_ = filepath.Walk(sessionsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		if _, ok := allowed[codexSessionCWD(path)]; !ok {
			return nil
		}
		matched = append(matched, fileInfo{path: path, modTime: info.ModTime()})
		return nil
	})

	if len(matched) == 0 {
		return "", fmt.Errorf("no Codex session found for %s", workingDir)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].modTime.After(matched[j].modTime)
	})

	c.cachedSession = matched[0].path
	return c.cachedSession, nil
}

func (c *CodexAdapter) Detect(workingDir string) bool {
	_, err := c.findSession(workingDir)
	return err == nil
}

func (c *CodexAdapter) LastActive(workingDir string) (time.Time, error) {
	sessionPath, err := c.findSession(workingDir)
	if err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(sessionPath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (c *CodexAdapter) Extract(workingDir string) (domain.SessionState, error) {
	sessionPath, err := c.findSession(workingDir)
	if err != nil {
		return domain.SessionState{}, err
	}

	file, err := os.Open(sessionPath)
	if err != nil {
		return domain.SessionState{}, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var turns []domain.Turn

	for {
		line, tooLarge, err := readLimitedLine(reader, codexMaxLineBytes)
		if err != nil {
			break
		}
		if tooLarge {
			continue
		}

		turn, ok := parseCodexTurn(line)
		if !ok {
			continue
		}
		turns = append(turns, turn)
	}

	var rawBytes int64
	if info, err := os.Stat(sessionPath); err == nil {
		rawBytes = info.Size()
	}

	return domain.SessionState{RecentTurns: selectContext(turns), RawBytes: rawBytes}, nil
}

func parseCodexTurn(line []byte) (domain.Turn, bool) {
	type contentItem struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type payload struct {
		Type    string        `json:"type"`
		Role    string        `json:"role"`
		Content []contentItem `json:"content"`
	}
	type codexLine struct {
		Type    string  `json:"type"`
		Payload payload `json:"payload"`
	}

	var entry codexLine
	if err := json.Unmarshal(line, &entry); err != nil {
		return domain.Turn{}, false
	}
	if entry.Type != "response_item" || entry.Payload.Type != "message" {
		return domain.Turn{}, false
	}
	if entry.Payload.Role != "user" && entry.Payload.Role != "assistant" {
		return domain.Turn{}, false
	}

	var parts []string
	for _, item := range entry.Payload.Content {
		if item.Text == "" {
			continue
		}
		if item.Type == "input_text" || item.Type == "output_text" || item.Type == "text" {
			parts = append(parts, item.Text)
		}
	}

	content := strings.TrimSpace(strings.Join(parts, "\n"))
	if content == "" {
		return domain.Turn{}, false
	}
	if len(content) > 1500 {
		content = content[:1500] + "\n... [Content truncated for brevity] ..."
	}

	return domain.Turn{Role: entry.Payload.Role, Content: content}, true
}

func (c *CodexAdapter) Name() string {
	return "Codex"
}
