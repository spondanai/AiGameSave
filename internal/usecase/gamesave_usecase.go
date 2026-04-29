package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spondanai/aigamesave/internal/adapters"
	"github.com/spondanai/aigamesave/internal/domain"
	"github.com/spondanai/aigamesave/internal/repository"
	"github.com/spondanai/aigamesave/pkg/clipboard"
	"github.com/spondanai/aigamesave/pkg/redaction"
)

const maxDiffBytes = 3000

// SaveGame coordinates the process of extracting context and saving it to YAML.
func SaveGame(workingDir string) error {
	return SaveGameWithAdapter(workingDir, "")
}

func SaveGameWithAdapter(workingDir, adapterName string) error {
	var adapter domain.HistoryExtractor
	var err error

	if adapterName != "" {
		adapter, err = adapters.DetectAdapterByName(workingDir, adapterName)
		if err != nil {
			return err
		}
	} else {
		adapter = adapters.DetectActiveAdapter(workingDir)
	}
	if adapter == nil {
		return fmt.Errorf("no supported AI CLI found in the current directory")
	}

	fmt.Printf("Detected AI CLI: %s\n", adapter.Name())

	state, err := adapter.Extract(workingDir)
	if err != nil {
		return fmt.Errorf("failed to extract history: %w", err)
	}

	files, err := repository.GetModifiedFiles(workingDir)
	if err == nil {
		state.GitVision = domain.RankFiles(files, state.RecentTurns)
		state.Diff = redaction.MaskSecrets(repository.GetDiff(workingDir, maxDiffBytes))
	} else {
		fmt.Println("Warning: Could not get git status (maybe not a repo?)")
	}

	for i := range state.RecentTurns {
		state.RecentTurns[i].Content = redaction.MaskSecrets(state.RecentTurns[i].Content)
	}

	state.ActiveFiles = filterExistingPaths(workingDir, domain.ExtractFilePaths(state.RecentTurns))

	if err := repository.SaveSession(workingDir, state); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	fmt.Println("Successfully saved session to .aigamesave.yaml")
	return nil
}

// filterExistingPaths returns only paths that resolve to real files under workingDir.
func filterExistingPaths(workingDir string, paths []string) []string {
	var result []string
	for _, p := range paths {
		abs := filepath.Join(workingDir, p)
		info, err := os.Stat(abs)
		if err == nil && !info.IsDir() {
			result = append(result, p)
		}
	}
	return result
}

// LoadGame reads the save file, formats a resume prompt, and copies it to the clipboard.
func LoadGame(workingDir string, toClipboard bool) (string, error) {
	state, err := repository.LoadSession(workingDir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("Resume Session:\n\n")

	sb.WriteString("## Recent context\n")
	for _, turn := range state.RecentTurns {
		fmt.Fprintf(&sb, "[%s]: %s\n", turn.Role, turn.Content)
	}

	if len(state.ActiveFiles) > 0 {
		sb.WriteString("\n## Active files (mentioned in conversation)\n")
		for _, p := range state.ActiveFiles {
			fmt.Fprintf(&sb, "- %s\n", p)
		}
	}

	if len(state.GitVision) > 0 {
		sb.WriteString("\n## Files being worked on (git status)\n")
		for _, file := range state.GitVision {
			fmt.Fprintf(&sb, "- %s [%s]\n", file.Path, file.Status)
		}
	}

	if state.Diff != "" {
		sb.WriteString("\n## Current diff (HEAD)\n```diff\n")
		sb.WriteString(state.Diff)
		sb.WriteString("\n```\n")
	}

	sb.WriteString("\nPlease continue the work based on this context.\n")
	prompt := sb.String()

	if toClipboard {
		if err := clipboard.WriteToClipboard(prompt); err != nil {
			fmt.Printf("Warning: Failed to copy to clipboard (%v). Returning text instead.\n", err)
			return prompt, nil
		}
		fmt.Println("Successfully loaded session and copied to clipboard!")
	}

	return prompt, nil
}
