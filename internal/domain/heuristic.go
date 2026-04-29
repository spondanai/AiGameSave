package domain

import (
	"regexp"
	"sort"
	"strings"
)

// filePathRe matches relative file paths that contain at least one "/" directory
// separator and end with a file extension. Requires the path to start with a
// letter so that version strings like "1.2.3/..." are not captured.
var filePathRe = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9_.%-]*/[a-zA-Z0-9_.%-][a-zA-Z0-9_./%-]*\.[a-zA-Z][a-zA-Z0-9]{0,11}`)

// ExtractFilePaths scans conversation turns for file path patterns and returns
// a deduplicated, ordered list. The caller is responsible for filtering paths
// that do not exist on disk.
func ExtractFilePaths(turns []Turn) []string {
	seen := make(map[string]struct{})
	var result []string

	for _, turn := range turns {
		for _, raw := range filePathRe.FindAllString(turn.Content, -1) {
			// Skip URL-like matches (e.g. github.com/foo/bar.go)
			if strings.Contains(raw, "://") || looksLikeDomain(raw) {
				continue
			}
			if _, ok := seen[raw]; !ok {
				seen[raw] = struct{}{}
				result = append(result, raw)
			}
		}
	}
	return result
}

// looksLikeDomain returns true when the first path component contains a dot,
// which indicates a hostname like "github.com" rather than a local directory.
func looksLikeDomain(path string) bool {
	host, _, ok := strings.Cut(path, "/")
	return ok && strings.Contains(host, ".")
}

const (
	maxFiles     = 10
	mentionBonus = 50
	mtimeMaxScore = 20
	mtimeStep    = 4
)

// RankFiles scores files by two heuristics (mention in turns > mtime),
// sorts by score descending, and returns at most maxFiles results.
func RankFiles(files []FileMetadata, recentTurns []Turn) []FileMetadata {
	// Build mtime rank scores: most recently modified file gets mtimeMaxScore,
	// each subsequent rank loses mtimeStep points, floor at 0.
	byMtime := make([]FileMetadata, len(files))
	copy(byMtime, files)
	sort.Slice(byMtime, func(i, j int) bool {
		return byMtime[i].ModTime.After(byMtime[j].ModTime)
	})
	mtimeScore := make(map[string]int, len(byMtime))
	for rank, f := range byMtime {
		score := max(0, mtimeMaxScore-rank*mtimeStep)
		mtimeScore[f.Path] = score
	}

	// Combine mtime score + mention bonus.
	for i := range files {
		score := mtimeScore[files[i].Path]
		for _, turn := range recentTurns {
			if strings.Contains(turn.Content, files[i].Path) {
				score += mentionBonus
				break
			}
		}
		files[i].Priority = score
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Priority > files[j].Priority
	})

	if len(files) > maxFiles {
		files = files[:maxFiles]
	}

	return files
}
