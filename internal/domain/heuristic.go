package domain

import (
	"strings"
)

// RankFiles assigns priority scores to files based on heuristics.
// In MVP: just simple ranking based on track/untrack or mention.
func RankFiles(files []FileMetadata, recentTurns []Turn) []FileMetadata {
	for i := range files {
		// Base score
		score := 0

		// Simple heuristic: If the file is mentioned in recent turns, boost score
		for _, turn := range recentTurns {
			if strings.Contains(turn.Content, files[i].Path) {
				score += 50
				break // only boost once per mention check
			}
		}

		files[i].Priority = score
	}

	// TODO: Sort files by priority (descending) if we need to limit the count.
	// For MVP, we might just return them all if it's not too many.

	return files
}
