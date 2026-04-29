package adapters

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
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
// Gemini CLI stores history globally in ~/.gemini/tmp/<user>/chats/, indexed by
// a projectHash (SHA-256 of the working directory) in each session's first line.
type GeminiAdapter struct {
	cachedChatFile string
}

func NewGeminiAdapter() *GeminiAdapter {
	return &GeminiAdapter{}
}

// projectHash computes the SHA-256 hex digest of a directory path,
// matching Gemini CLI's own session-to-project correlation.
func projectHash(dir string) string {
	sum := sha256.Sum256([]byte(dir))
	return hex.EncodeToString(sum[:])
}

// candidateHashes returns hashes for workingDir and each parent up to home.
// This handles the case where Gemini CLI was launched from a parent directory.
func candidateHashes(workingDir string) map[string]struct{} {
	homeDir, _ := os.UserHomeDir()
	hashes := make(map[string]struct{})
	dir := workingDir
	for {
		hashes[projectHash(dir)] = struct{}{}
		if dir == homeDir || dir == "/" || dir == "." {
			break
		}
		dir = filepath.Dir(dir)
	}
	return hashes
}

type geminiMeta struct {
	ProjectHash string `json:"projectHash"`
}

// readProjectHash reads only the first line of a session file to extract projectHash.
func readProjectHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		var m geminiMeta
		if err := json.Unmarshal(scanner.Bytes(), &m); err == nil {
			return m.ProjectHash
		}
	}
	return ""
}

// findChatFile searches ~/.gemini/tmp/*/chats/ for the most recently modified
// session whose projectHash matches workingDir or any parent directory.
func (g *GeminiAdapter) findChatFile(workingDir string) (string, error) {
	if g.cachedChatFile != "" {
		return g.cachedChatFile, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	allowed := candidateHashes(workingDir)
	tmpDir := filepath.Join(homeDir, ".gemini", "tmp")

	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var matched []fileInfo

	_ = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".jsonl") || filepath.Base(filepath.Dir(path)) != "chats" {
			return nil
		}
		hash := readProjectHash(path)
		if _, ok := allowed[hash]; ok {
			matched = append(matched, fileInfo{path: path, modTime: info.ModTime()})
		}
		return nil
	})

	if len(matched) == 0 {
		return "", fmt.Errorf("no gemini session found for %s", workingDir)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].modTime.After(matched[j].modTime)
	})

	g.cachedChatFile = matched[0].path
	return g.cachedChatFile, nil
}

// Detect returns true only if a Gemini session correlates with workingDir or a parent.
func (g *GeminiAdapter) Detect(workingDir string) bool {
	_, err := g.findChatFile(workingDir)
	return err == nil
}

func (g *GeminiAdapter) LastActive(workingDir string) (time.Time, error) {
	historyPath, err := g.findChatFile(workingDir)
	if err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(historyPath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// Extract parses the correlated Gemini CLI session file.
func (g *GeminiAdapter) Extract(workingDir string) (domain.SessionState, error) {
	historyPath, err := g.findChatFile(workingDir)
	if err != nil {
		return domain.SessionState{}, fmt.Errorf("failed to find Gemini chat history: %w", err)
	}

	file, err := os.Open(historyPath)
	if err != nil {
		return domain.SessionState{}, err
	}
	defer file.Close()

	var turns []domain.Turn
	reader := bufio.NewReader(file)

	type GeminiLine struct {
		Type    string `json:"type"`
		Content any    `json:"content"`
	}

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

		// Skip metadata lines (first line and any state updates)
		lineStr := string(fullLine)
		if strings.HasPrefix(lineStr, `{"$set"`) || strings.Contains(lineStr, `"sessionId"`) || strings.Contains(lineStr, `"projectHash"`) {
			continue
		}

		var gl GeminiLine
		if err := json.Unmarshal(fullLine, &gl); err == nil {
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

	maxTurns := 6
	if len(turns) > maxTurns {
		turns = turns[len(turns)-maxTurns:]
	}

	return domain.SessionState{RecentTurns: turns}, nil
}

func (g *GeminiAdapter) Name() string {
	return "Gemini CLI"
}
