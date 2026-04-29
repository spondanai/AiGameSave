package adapters

import (
	"unicode/utf8"

	"github.com/spondanai/aigamesave/internal/domain"
)

const (
	// minSubstantialRunes is the minimum character count for a user turn to be
	// treated as a meaningful instruction rather than a short acknowledgment.
	minSubstantialRunes = 10

	// maxTailTurns is the number of turns to include after the anchor.
	// We take the LAST N (not first N) so the context reflects the most recent work.
	maxTailTurns = 5

	// fallbackTurns is used when no substantial user anchor is found.
	fallbackTurns = 6
)

// selectContext picks context that preserves the original user instruction.
//
// It walks backward to find the last substantial user message (the "anchor"),
// then returns that anchor plus the most recent turns that followed it.
// This prevents agentic self-talk from pushing the user's original goal
// out of the context window.
func selectContext(turns []domain.Turn) []domain.Turn {
	anchorIdx := -1
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].Role == "user" && isSubstantial(turns[i].Content) {
			anchorIdx = i
			break
		}
	}

	if anchorIdx == -1 {
		if len(turns) > fallbackTurns {
			return turns[len(turns)-fallbackTurns:]
		}
		return turns
	}

	tail := turns[anchorIdx+1:]
	if len(tail) > maxTailTurns {
		tail = tail[len(tail)-maxTailTurns:]
	}

	result := make([]domain.Turn, 0, 1+len(tail))
	result = append(result, turns[anchorIdx])
	result = append(result, tail...)
	return result
}

// isSubstantial returns true if the content looks like a real instruction
// rather than a short filler acknowledgment ("โอเค", "ok", "ลองรันดู").
func isSubstantial(content string) bool {
	return utf8.RuneCountInString(content) >= minSubstantialRunes
}
