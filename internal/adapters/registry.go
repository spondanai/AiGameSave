package adapters

import (
	"fmt"
	"strings"
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
	registry = append(registry, NewAiderAdapter(), NewClaudeAdapter(), NewGeminiAdapter(), NewCodexAdapter(), NewCopilotAdapter(), NewClineAdapter())
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

func DetectAdapterByName(workingDir, name string) (domain.HistoryExtractor, error) {
	wanted := normalizeAdapterName(name)
	for _, adapter := range registry {
		if !adapterNameMatches(adapter.Name(), wanted) {
			continue
		}
		if !adapter.Detect(workingDir) {
			return nil, fmt.Errorf("%s was requested, but no matching session was found in this directory", adapter.Name())
		}
		return adapter, nil
	}
	return nil, fmt.Errorf("unknown adapter %q (available: %s)", name, strings.Join(AdapterNames(), ", "))
}

func AdapterNames() []string {
	names := make([]string, 0, len(registry))
	for _, adapter := range registry {
		names = append(names, adapter.Name())
	}
	return names
}

func normalizeAdapterName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	return replacer.Replace(name)
}

func adapterNameMatches(adapterName, wanted string) bool {
	normalized := normalizeAdapterName(adapterName)
	return normalized == wanted || strings.HasPrefix(normalized, wanted) || strings.HasSuffix(normalized, wanted)
}
