package adapters

import (
	"github.com/spondanai/AiGameSave/internal/domain"
)

// Registry holds all registered AI CLI adapters.
var registry []domain.HistoryExtractor

func init() {
	// Register known adapters here.
	// In the future, we can iterate over these dynamically.
	registry = append(registry, NewAiderAdapter(), NewClaudeAdapter(), NewGeminiAdapter())
}

// DetectActiveAdapter iterates through the registry and returns the first matching adapter.
func DetectActiveAdapter(workingDir string) domain.HistoryExtractor {
	for _, adapter := range registry {
		if adapter.Detect(workingDir) {
			return adapter
		}
	}
	return nil
}


