package repository

import (
	"os/exec"
	"strings"

	"github.com/mrporing/aigamesave/internal/domain"
)

// GetModifiedFiles runs `git status --porcelain` and returns the file metadata.
func GetModifiedFiles(workingDir string) ([]domain.FileMetadata, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = workingDir

	out, err := cmd.Output()
	if err != nil {
		// Possibly not a git repository
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var files []domain.FileMetadata

	for _, line := range lines {
		if len(line) < 4 {
			continue // Skip empty or malformed lines
		}
		
		status := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])

		files = append(files, domain.FileMetadata{
			Path:   path,
			Status: status,
		})
	}

	return files, nil
}
