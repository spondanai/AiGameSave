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

// GetDiff returns a combined diff: `git diff HEAD` for tracked changes plus
// the content of untracked files (from `git ls-files --others --exclude-standard`).
// Total output is truncated to maxBytes.
func GetDiff(workingDir string, maxBytes int) string {
	var sb strings.Builder

	// Tracked changes
	if out, err := runGit(workingDir, "diff", "HEAD"); err == nil && len(out) > 0 {
		sb.Write(out)
	}

	// Untracked files — include their full content so new-file sessions are not context-blind
	if out, err := runGit(workingDir, "ls-files", "--others", "--exclude-standard"); err == nil {
		for rel := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
			if rel == "" {
				continue
			}
			content, err := os.ReadFile(filepath.Join(workingDir, rel))
			if err != nil {
				continue
			}
			sb.WriteString("\n--- untracked: " + rel + " ---\n")
			sb.Write(content)
		}
	}

	diff := sb.String()
	if len(diff) == 0 {
		return ""
	}
	if len(diff) > maxBytes {
		diff = diff[:maxBytes] + "\n... [diff truncated] ..."
	}
	return diff
}

func runGit(workingDir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workingDir
	return cmd.Output()
}
