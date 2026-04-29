package usecase

import (
	"fmt"
	"strings"

	"github.com/spondanai/AiGameSave/internal/adapters"
	"github.com/spondanai/AiGameSave/internal/domain"
	"github.com/spondanai/AiGameSave/internal/repository"
	"github.com/spondanai/AiGameSave/pkg/redaction"
	"github.com/spondanai/AiGameSave/pkg/clipboard"
)

// SaveGame coordinates the process of extracting context and saving it to YAML.
func SaveGame(workingDir string) error {
	adapter := adapters.DetectActiveAdapter(workingDir)
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
		// Rank and add git vision if available
		state.GitVision = domain.RankFiles(files, state.RecentTurns)
	} else {
		fmt.Println("Warning: Could not get git status (maybe not a repo?)")
	}

	// Redact secrets in the extracted state
	for i := range state.RecentTurns {
		state.RecentTurns[i].Content = redaction.MaskSecrets(state.RecentTurns[i].Content)
	}

	err = repository.SaveSession(workingDir, state)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	fmt.Println("Successfully saved session to .aigamesave.yaml")
	return nil
}

// LoadGame reads the save file, formats a prompt, and copies it to the clipboard or returns it.
func LoadGame(workingDir string, toClipboard bool) (string, error) {
	state, err := repository.LoadSession(workingDir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("Resume Session:\n\n")
	
	sb.WriteString("Recent context:\n")
	for _, turn := range state.RecentTurns {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", turn.Role, turn.Content))
	}
	
	sb.WriteString("\nCurrent files of interest:\n")
	for _, file := range state.GitVision {
		sb.WriteString(fmt.Sprintf("- %s (Status: %s)\n", file.Path, file.Status))
	}
	
	sb.WriteString("\nPlease continue the work based on this context.\n")
	prompt := sb.String()

	if toClipboard {
		err := clipboard.WriteToClipboard(prompt)
		if err != nil {
			fmt.Printf("Warning: Failed to copy to clipboard (%v). Returning text instead.\n", err)
			return prompt, nil // Return the text anyway
		}
		fmt.Println("Successfully loaded session and copied to clipboard!")
	}

	return prompt, nil
}
