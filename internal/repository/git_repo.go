package repository

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spondanai/aigamesave/internal/domain"
)

// GetModifiedFiles runs `git status --porcelain` and returns file metadata including mtime.
func GetModifiedFiles(workingDir string) ([]domain.FileMetadata, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = workingDir

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var files []domain.FileMetadata

	for _, line := range lines {
		if len(line) < 4 {
			continue
		}

		status := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])

		meta := domain.FileMetadata{
			Path:   path,
			Status: status,
		}

		if info, err := os.Stat(filepath.Join(workingDir, path)); err == nil {
			meta.ModTime = info.ModTime()
		}

		files = append(files, meta)
	}

	return files, nil
}
