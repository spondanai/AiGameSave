// Package ignore reads .aigamesaveignore and decides which paths to exclude
// from the saved context. The syntax is a subset of .gitignore:
//
//   - Blank lines and lines starting with '#' are ignored.
//   - Patterns with no '/' match against the file's basename only.
//   - Patterns ending with '/' or '/**' match a directory and everything inside it.
//   - '**/' prefix means "match in any directory" (same as no-slash rule).
//   - All other patterns are matched against the full forward-slash path.
package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const IgnoreFileName = ".aigamesaveignore"

// Rules holds the parsed patterns loaded from an ignore file.
type Rules struct {
	patterns []string
}

// Load reads .aigamesaveignore from workingDir. Returns empty Rules (not an
// error) when the file does not exist — callers need not special-case that.
func Load(workingDir string) (*Rules, error) {
	f, err := os.Open(filepath.Join(workingDir, IgnoreFileName))
	if os.IsNotExist(err) {
		return &Rules{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var patterns []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, filepath.ToSlash(line))
	}
	return &Rules{patterns: patterns}, sc.Err()
}

// Ignored returns true if path should be excluded from the saved context.
// path must use forward slashes (as stored in ActiveFiles / GitVision).
func (r *Rules) Ignored(path string) bool {
	path = filepath.ToSlash(path)
	for _, p := range r.patterns {
		if matchPattern(p, path) {
			return true
		}
	}
	return false
}

// matchPattern checks whether a single pattern matches a forward-slash path.
func matchPattern(pattern, path string) bool {
	// Directory patterns: "vendor/" or "vendor/**"
	if dir, ok := strings.CutSuffix(pattern, "/"); ok {
		return path == dir || strings.HasPrefix(path, dir+"/")
	}
	if dir, ok := strings.CutSuffix(pattern, "/**"); ok {
		return path == dir || strings.HasPrefix(path, dir+"/")
	}

	// "**/<pattern>" — strip the prefix, fall through to basename match.
	pattern = strings.TrimPrefix(pattern, "**/")


	// Pattern with "/" → match against the full path.
	if strings.Contains(pattern, "/") {
		matched, _ := filepath.Match(pattern, path)
		return matched
	}

	// No "/" → match against the basename only so "*.pb.go" hits
	// "internal/foo.pb.go" without the caller needing to write "*/**/*.pb.go".
	matched, _ := filepath.Match(pattern, filepath.Base(path))
	return matched
}
