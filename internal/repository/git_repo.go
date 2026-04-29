package repository

import (
	"bytes"
	"io"
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
		if len(out) > maxBytes {
			sb.Write(out[:maxBytes])
		} else {
			sb.Write(out)
		}
	}

	// Untracked files — include their full content so new-file sessions are not context-blind
	if sb.Len() < maxBytes {
		if out, err := runGit(workingDir, "ls-files", "--others", "--exclude-standard"); err == nil {
			for rel := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
				if rel == "" {
					continue
				}
				if sb.Len() >= maxBytes {
					break
				}

				fPath := filepath.Join(workingDir, rel)
				info, err := os.Stat(fPath)
				if err != nil || info.IsDir() {
					continue
				}

				// Skip large untracked files completely (e.g. > 1MB)
				if info.Size() > 1024*1024 {
					sb.WriteString("\n--- untracked: " + rel + " [skipped: too large] ---\n")
					continue
				}

				f, err := os.Open(fPath)
				if err != nil {
					continue
				}

				remaining := maxBytes - sb.Len()
				if remaining <= 0 {
					f.Close()
					break
				}

				// We want to write "\n--- untracked: " + rel + " ---\n"
				header := "\n--- untracked: " + rel + " ---\n"
				remaining -= len(header)
				if remaining <= 0 {
					f.Close()
					break
				}

				content, err := io.ReadAll(io.LimitReader(f, int64(remaining)))
				f.Close()

				if err == nil {
					sb.WriteString(header)
					// Basic check: if it has null bytes, treat as binary and skip printing
					if bytes.IndexByte(content, 0) != -1 {
						sb.WriteString("[skipped: binary file]\n")
					} else {
						sb.Write(content)
					}
				}
			}
		}
	}

	diff := sb.String()
	if len(diff) == 0 {
		return ""
	}
	if len(diff) >= maxBytes {
		// Because we cap appending, it might be exactly maxBytes
		return diff[:maxBytes] + "\n... [diff truncated] ..."
	}
	return diff
}

func runGit(workingDir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workingDir
	return cmd.Output()
}
