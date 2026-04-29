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

// GetDiff runs `git diff HEAD` and returns the output truncated to maxBytes.
// Returns an empty string if not a git repo or there is nothing to diff.
func GetDiff(workingDir string, maxBytes int) string {
	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = workingDir

	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return ""
	}

	diff := string(out)
	if len(diff) > maxBytes {
		diff = diff[:maxBytes] + "\n... [diff truncated] ..."
	}
	return diff
}
