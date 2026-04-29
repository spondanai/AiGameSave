package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/spondanai/aigamesave/internal/domain"
)

const SaveFileName = ".aigamesave.yaml"

// SaveSession writes the extracted session state to `.aigamesave.yaml`.
func SaveSession(workingDir string, state domain.SessionState) error {
	savePath := filepath.Join(workingDir, SaveFileName)

	data, err := yaml.Marshal(&state)
	if err != nil {
		return fmt.Errorf("failed to marshal state to YAML: %w", err)
	}

	err = os.WriteFile(savePath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write save file: %w", err)
	}

	if err := ensureGitIgnore(workingDir); err != nil {
		fmt.Printf("Warning: could not update .gitignore: %v\n", err)
	}

	return nil
}

// ensureGitIgnore appends `.aigamesave.yaml` to `.gitignore` if it's not already there.
func ensureGitIgnore(workingDir string) error {
	ignorePath := filepath.Join(workingDir, ".gitignore")
	content, err := os.ReadFile(ignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(ignorePath, []byte(SaveFileName+"\n"), 0644)
		}
		return err
	}

	if strings.Contains(string(content), SaveFileName) {
		return nil
	}

	f, err := os.OpenFile(ignorePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if !strings.HasSuffix(string(content), "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(SaveFileName + "\n")
	return err
}

// LoadSession reads `.aigamesave.yaml` and parses the state.
func LoadSession(workingDir string) (domain.SessionState, error) {
	var state domain.SessionState
	savePath := filepath.Join(workingDir, SaveFileName)

	data, err := os.ReadFile(savePath)
	if err != nil {
		return state, fmt.Errorf("save file not found or cannot be read: %w", err)
	}

	err = yaml.Unmarshal(data, &state)
	if err != nil {
		return state, fmt.Errorf("failed to parse save file: %w", err)
	}

	return state, nil
}
