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

// noisyDiffFiles are auto-generated or dependency-managed files whose diffs
// never help an AI resume a session — they only consume the byte budget.
var noisyDiffFiles = map[string]bool{
	"go.sum":            true,
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"Cargo.lock":        true,
	"Gemfile.lock":      true,
	"poetry.lock":       true,
	"composer.lock":     true,
	"Pipfile.lock":      true,
}

// isNoisyFile reports whether path should be excluded from the diff.
func isNoisyFile(path string) bool {
	base := filepath.Base(path)
	return noisyDiffFiles[base] ||
		strings.HasSuffix(base, ".min.js") ||
		strings.HasSuffix(base, ".min.css")
}

// buildDiffExcludes returns :(exclude) pathspecs for all noisy file names.
func buildDiffExcludes() []string {
	out := make([]string, 0, len(noisyDiffFiles))
	for name := range noisyDiffFiles {
		out = append(out, ":(exclude)"+name)
	}
	return out
}

// GetDiff returns a combined diff: `git diff HEAD` for tracked changes plus
// the content of untracked files (from `git ls-files --others --exclude-standard`).
// Auto-generated files (lock files, minified assets) are excluded so that the
// byte budget is spent on meaningful changes. Total output is truncated to maxBytes.
func GetDiff(workingDir string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}

	var sb strings.Builder
	truncated := false

	// Tracked changes — exclude noisy files at the git level so they don't
	// consume the byte budget before real source changes get a chance.
	diffArgs := append([]string{"diff", "HEAD", "--"}, buildDiffExcludes()...)
	if out, wasTruncated, err := runGitLimited(workingDir, maxBytes, diffArgs...); err == nil && len(out) > 0 {
		sb.Write(out)
		truncated = wasTruncated
	}

	// Untracked files — include their full content so new-file sessions are not context-blind
	if !truncated && sb.Len() < maxBytes {
		if out, err := runGit(workingDir, "ls-files", "--others", "--exclude-standard"); err == nil {
			for rel := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
				if rel == "" {
					continue
				}
				if sb.Len() >= maxBytes {
					break
				}

				if isNoisyFile(rel) {
					continue
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
	if truncated || len(diff) >= maxBytes {
		return diff[:maxBytes] + "\n... [diff truncated] ..."
	}
	return diff
}

func runGitLimited(workingDir string, maxBytes int, args ...string) ([]byte, bool, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workingDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, err
	}
	if err := cmd.Start(); err != nil {
		return nil, false, err
	}

	out, readErr := io.ReadAll(io.LimitReader(stdout, int64(maxBytes+1)))
	wasTruncated := len(out) > maxBytes
	if wasTruncated && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if readErr != nil {
		return nil, false, readErr
	}
	if waitErr != nil && !wasTruncated {
		return nil, false, waitErr
	}

	if wasTruncated {
		return out[:maxBytes], true, nil
	}
	return out, false, nil
}

func runGit(workingDir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workingDir
	return cmd.Output()
}
