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

// GeminiAdapter is the adapter for Gemini CLI.
// Gemini CLI stores history globally in ~/.gemini/tmp/*/chats/, not per-project.
type GeminiAdapter struct {
	cachedChatFile string
}

func NewGeminiAdapter() *GeminiAdapter {
	return &GeminiAdapter{}
}

// getLatestChatFile finds the most recently modified .jsonl file across all ~/.gemini/tmp/*/chats/.
// Result is cached to avoid redundant filesystem walks between Detect and Extract.
func (g *GeminiAdapter) getLatestChatFile() (string, error) {
	if g.cachedChatFile != "" {
		return g.cachedChatFile, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	tmpDir := filepath.Join(homeDir, ".gemini", "tmp")

	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var files []fileInfo

	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // ignore permission errors
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".jsonl") && filepath.Base(filepath.Dir(path)) == "chats" {
			files = append(files, fileInfo{path: path, modTime: info.ModTime()})
		}
		return nil
	})

	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("no gemini chat files found in %s", tmpDir)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	g.cachedChatFile = files[0].path
	return g.cachedChatFile, nil
}

// Detect checks if Gemini CLI has any recent chat history.
// Note: Gemini stores history globally, so this is not scoped to workingDir.
func (g *GeminiAdapter) Detect(workingDir string) bool {
	_, err := g.getLatestChatFile()
	return err == nil
}

// Extract parses the Gemini CLI history file.
func (g *GeminiAdapter) Extract(workingDir string) (domain.SessionState, error) {
	historyPath, err := g.getLatestChatFile()
	if err != nil {
		return domain.SessionState{}, fmt.Errorf("failed to find Gemini chat history: %w", err)
	}

	file, err := os.Open(historyPath)
	if err != nil {
		return domain.SessionState{}, err
	}
	defer file.Close()

	var turns []domain.Turn
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	type GeminiLine struct {
		Type    string `json:"type"`
		Content any    `json:"content"`
	}

	for scanner.Scan() {
		line := scanner.Bytes()

		if strings.HasPrefix(string(line), `{"$set"`) || strings.Contains(string(line), `"sessionId"`) {
			continue
		}

		var gl GeminiLine
		if err := json.Unmarshal(line, &gl); err == nil {
			role := gl.Type
			if role == "" {
				continue
			}

			var contentStr string
			switch v := gl.Content.(type) {
			case string:
				contentStr = v
			case []any:
				for _, item := range v {
					if obj, ok := item.(map[string]any); ok {
						if text, ok := obj["text"].(string); ok {
							contentStr += text + "\n"
						}
					}
				}
			}

			contentStr = strings.TrimSpace(contentStr)
			if contentStr == "" {
				continue
			}

			if len(contentStr) > 1500 {
				contentStr = contentStr[:1500] + "\n... [Content truncated for brevity] ..."
			}
			turns = append(turns, domain.Turn{Role: role, Content: contentStr})
		}
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

func (g *GeminiAdapter) Name() string {
	return "Gemini CLI"
}
