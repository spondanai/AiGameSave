package domain

import (
	"sort"
	"strings"
)

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
