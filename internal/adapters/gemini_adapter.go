package adapters

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spondanai/AiGameSave/internal/domain"
)

// GeminiAdapter is the adapter for Gemini CLI.
type GeminiAdapter struct{}

func NewGeminiAdapter() *GeminiAdapter {
	return &GeminiAdapter{}
}

// getLatestChatFile finds the most recently modified .jsonl file across all ~/.gemini/tmp/*/chats/
func (g *GeminiAdapter) getLatestChatFile(workingDir string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	tmpDir := filepath.Join(homeDir, ".gemini", "tmp")
	var files []string

	// Walk through all directories in ~/.gemini/tmp
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // ignore permission errors
		}
		
		// Only look for .jsonl files in a "chats" directory
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".jsonl") && filepath.Base(filepath.Dir(path)) == "chats" {
			files = append(files, path)
		}
		return nil
	})

	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("no gemini chat files found in %s", tmpDir)
	}

	// Sort files by modification time (descending)
	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return files[0], nil
}

// Detect checks if Gemini CLI is active by looking for a recent chat file.
// Since Gemini CLI history is stored globally, we consider it "detected" if a recent file exists.
func (g *GeminiAdapter) Detect(workingDir string) bool {
	_, err := g.getLatestChatFile(workingDir)
	return err == nil
}


// Extract parses the Gemini CLI history file.
// It extracts the last few turns and truncates long messages.
func (g *GeminiAdapter) Extract(workingDir string) (domain.SessionState, error) {
	historyPath, err := g.getLatestChatFile(workingDir)
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

	// Represents the structure for Gemini CLI turns.
	type ContentItem struct {
		Text string `json:"text"`
	}
	type GeminiLine struct {
		Type    string      `json:"type"` // "user" or "gemini"
		Content interface{} `json:"content"` // can be string or []ContentItem
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		
		// Skip metadata or state lines
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
			case []interface{}:
				// Handle array of objects (like the user message)
				for _, item := range v {
					if obj, ok := item.(map[string]interface{}); ok {
						if text, ok := obj["text"].(string); ok {
							contentStr += text + "\n"
						}
					}
				}
			}

			contentStr = strings.TrimSpace(contentStr)
			if contentStr != "" {
				// Prevent extremely long outputs from eating up context space
				if len(contentStr) > 1500 {
					contentStr = contentStr[:1500] + "\n... [Content truncated for brevity] ..."
				}
				turns = append(turns, domain.Turn{Role: role, Content: contentStr})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return domain.SessionState{}, err
	}

	// Keep only the last 6 turns (3 pairs)
	maxTurns := 6
	if len(turns) > maxTurns {
		turns = turns[len(turns)-maxTurns:]
	}

	return domain.SessionState{
		RecentTurns: turns,
	}, nil
}

func (g *GeminiAdapter) Name() string {
	return "Gemini CLI"
}

