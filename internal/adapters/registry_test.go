package adapters

import (
	"testing"
	"time"

	"github.com/spondanai/aigamesave/internal/domain"
)

type fakeAdapter struct {
	name       string
	detected   bool
	lastActive time.Time
	lastErr    error
}

func (f fakeAdapter) Detect(workingDir string) bool {
	return f.detected
}

func (f fakeAdapter) Extract(workingDir string) (domain.SessionState, error) {
	return domain.SessionState{}, nil
}

func (f fakeAdapter) Name() string {
	return f.name
}

func (f fakeAdapter) LastActive(workingDir string) (time.Time, error) {
	return f.lastActive, f.lastErr
}

func withRegistry(t *testing.T, adapters ...domain.HistoryExtractor) {
	t.Helper()

	original := registry
	registry = adapters
	t.Cleanup(func() {
		registry = original
	})
}

func TestDetectActiveAdapterChoosesLatestSession(t *testing.T) {
	now := time.Now()
	withRegistry(t,
		fakeAdapter{name: "Claude Code", detected: true, lastActive: now.Add(-time.Hour)},
		fakeAdapter{name: "Gemini CLI", detected: true, lastActive: now},
	)

	selected := DetectActiveAdapter(t.TempDir())
	if selected == nil {
		t.Fatal("expected an adapter")
	}
	if selected.Name() != "Gemini CLI" {
		t.Fatalf("expected Gemini CLI, got %s", selected.Name())
	}
}

func TestDetectActiveAdapterChoosesClaudeWhenNewer(t *testing.T) {
	now := time.Now()
	withRegistry(t,
		fakeAdapter{name: "Claude Code", detected: true, lastActive: now},
		fakeAdapter{name: "Gemini CLI", detected: true, lastActive: now.Add(-time.Hour)},
	)

	selected := DetectActiveAdapter(t.TempDir())
	if selected == nil {
		t.Fatal("expected an adapter")
	}
	if selected.Name() != "Claude Code" {
		t.Fatalf("expected Claude Code, got %s", selected.Name())
	}
}

func TestDetectActiveAdapterIgnoresUndetectedAdapters(t *testing.T) {
	now := time.Now()
	withRegistry(t,
		fakeAdapter{name: "Claude Code", detected: false, lastActive: now.Add(time.Hour)},
		fakeAdapter{name: "Gemini CLI", detected: true, lastActive: now},
	)

	selected := DetectActiveAdapter(t.TempDir())
	if selected == nil {
		t.Fatal("expected an adapter")
	}
	if selected.Name() != "Gemini CLI" {
		t.Fatalf("expected Gemini CLI, got %s", selected.Name())
	}
}
