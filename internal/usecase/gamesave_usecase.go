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
	"github.com/spondanai/aigamesave/pkg/ignore"
	"github.com/spondanai/aigamesave/pkg/redaction"
)

const maxDiffBytes = 3000

// SaveOptions controls optional behaviour of the save pipeline.
type SaveOptions struct {
	// AdapterName forces a specific AI adapter (empty = auto-detect).
	AdapterName string
	// NoDiff skips embedding the git diff in the saved state.
	NoDiff bool
}

// SaveGame coordinates the process of extracting context and saving it to YAML.
func SaveGame(workingDir string) error {
	return SaveGameWithOptions(workingDir, SaveOptions{})
}

// SaveGameWithAdapter is kept for backwards compatibility.
func SaveGameWithAdapter(workingDir, adapterName string) error {
	return SaveGameWithOptions(workingDir, SaveOptions{AdapterName: adapterName})
}

func SaveGameWithOptions(workingDir string, opts SaveOptions) error {
	var adapter domain.HistoryExtractor
	var err error

	if opts.AdapterName != "" {
		adapter, err = adapters.DetectAdapterByName(workingDir, opts.AdapterName)
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
		if !opts.NoDiff {
			state.Diff = redaction.MaskSecrets(repository.GetDiff(workingDir, maxDiffBytes))
		}
	} else {
		fmt.Println("Warning: Could not get git status (maybe not a repo?)")
	}

	for i := range state.RecentTurns {
		state.RecentTurns[i].Content = redaction.MaskSecrets(state.RecentTurns[i].Content)
	}

	state.ActiveFiles = filterExistingPaths(workingDir, domain.ExtractFilePaths(state.RecentTurns))

	ignoreRules, _ := ignore.Load(workingDir)
	state.GitVision = filterIgnored(state.GitVision, func(f domain.FileMetadata) string { return f.Path }, ignoreRules)
	state.ActiveFiles = filterIgnoredStrings(state.ActiveFiles, ignoreRules)

	if err := repository.SaveSession(workingDir, state); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	fmt.Println("Successfully saved session to .aigamesave.yaml")
	printTokenSavings(state)
	return nil
}

// printTokenSavings reports how many tokens the resume prompt saves vs the raw
// session file. Uses 4 bytes-per-token as a conservative approximation.
func printTokenSavings(state domain.SessionState) {
	if state.RawBytes == 0 {
		return
	}
	resumeBytes := int64(len([]byte(formatResumePrompt(state))))
	rawTokens := state.RawBytes / 4
	resumeTokens := resumeBytes / 4
	saved := rawTokens - resumeTokens
	if saved <= 0 {
		return
	}
	pct := (state.RawBytes - resumeBytes) * 100 / state.RawBytes
	fmt.Printf("Token savings: ~%s tokens saved vs raw session (%d%% reduction)\n",
		formatTokenCount(saved), pct)
}

func formatTokenCount(n int64) string {
	if n >= 1000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%d", n)
}

// formatResumePrompt builds the same resume prompt that LoadGame would produce,
// so we can measure its size at save time.
func formatResumePrompt(state domain.SessionState) string {
	var sb strings.Builder
	sb.WriteString("Resume Session:\n\n## Recent context\n")
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
	return sb.String()
}

// filterIgnored removes FileMetadata entries whose paths match ignoreRules.
func filterIgnored(files []domain.FileMetadata, key func(domain.FileMetadata) string, rules *ignore.Rules) []domain.FileMetadata {
	out := files[:0:0]
	for _, f := range files {
		if !rules.Ignored(key(f)) {
			out = append(out, f)
		}
	}
	return out
}

// filterIgnoredStrings removes string paths that match ignoreRules.
func filterIgnoredStrings(paths []string, rules *ignore.Rules) []string {
	out := paths[:0:0]
	for _, p := range paths {
		if !rules.Ignored(p) {
			out = append(out, p)
		}
	}
	return out
}

// filterExistingPaths returns only paths that resolve to real files under workingDir.
// Paths are stored with forward slashes; filepath.FromSlash converts them to the
// OS-native separator before os.Stat so the check works correctly on Windows.
func filterExistingPaths(workingDir string, paths []string) []string {
	var result []string
	for _, p := range paths {
		abs := filepath.Join(workingDir, filepath.FromSlash(p))
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
