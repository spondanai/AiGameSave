package domain

import (
	"sort"
	"strings"
)

// RankFiles assigns priority scores to files based on heuristics and returns them sorted.
func RankFiles(files []FileMetadata, recentTurns []Turn) []FileMetadata {
	for i := range files {
		score := 0
		for _, turn := range recentTurns {
			if strings.Contains(turn.Content, files[i].Path) {
				score += 50
				break
			}
		}
		files[i].Priority = score
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Priority > files[j].Priority
	})

	return files
}
