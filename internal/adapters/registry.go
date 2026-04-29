package adapters

import (
	"time"

	"github.com/spondanai/aigamesave/internal/domain"
)

// Registry holds all registered AI CLI adapters.
var registry []domain.HistoryExtractor

type activeAdapter interface {
	LastActive(workingDir string) (time.Time, error)
}

func init() {
	// Register known adapters here.
	// In the future, we can iterate over these dynamically.
	registry = append(registry, NewAiderAdapter(), NewClaudeAdapter(), NewGeminiAdapter(), NewCodexAdapter())
}

// DetectActiveAdapter chooses the matching adapter with the most recently
// updated session, so concurrent use of multiple AI CLIs resumes the right one.
func DetectActiveAdapter(workingDir string) domain.HistoryExtractor {
	var selected domain.HistoryExtractor
	var selectedAt time.Time

	for _, adapter := range registry {
		if !adapter.Detect(workingDir) {
			continue
		}

		active, ok := adapter.(activeAdapter)
		if !ok {
			if selected == nil {
				selected = adapter
			}
			continue
		}

		lastActive, err := active.LastActive(workingDir)
		if err != nil {
			continue
		}
		if selected == nil || lastActive.After(selectedAt) {
			selected = adapter
			selectedAt = lastActive
		}
	}

	return selected
}
