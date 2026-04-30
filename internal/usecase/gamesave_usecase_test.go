package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spondanai/aigamesave/internal/domain"
	"github.com/spondanai/aigamesave/pkg/ignore"
)

// ---- formatTokenCount ----

func TestFormatTokenCount_BelowThousand(t *testing.T) {
	if got := formatTokenCount(999); got != "999" {
		t.Errorf("want 999 got %q", got)
	}
}

func TestFormatTokenCount_ExactlyThousand(t *testing.T) {
	if got := formatTokenCount(1000); got != "1k" {
		t.Errorf("want 1k got %q", got)
	}
}

func TestFormatTokenCount_AboveThousand(t *testing.T) {
	if got := formatTokenCount(4500); got != "4k" {
		t.Errorf("want 4k got %q", got)
	}
}

// ---- formatResumePrompt ----

func TestFormatResumePrompt_ContainsRecentTurns(t *testing.T) {
	state := domain.SessionState{
		RecentTurns: []domain.Turn{
			{Role: "user", Content: "Fix the bug"},
			{Role: "assistant", Content: "Done."},
		},
	}
	out := formatResumePrompt(state)
	if !strings.Contains(out, "[user]: Fix the bug") {
		t.Errorf("missing user turn: %q", out)
	}
	if !strings.Contains(out, "[assistant]: Done.") {
		t.Errorf("missing assistant turn: %q", out)
	}
}

func TestFormatResumePrompt_OmitsPendingWhenEmpty(t *testing.T) {
	state := domain.SessionState{
		RecentTurns: []domain.Turn{{Role: "user", Content: "hi"}},
	}
	out := formatResumePrompt(state)
	if strings.Contains(out, "Pending plan") {
		t.Errorf("should not contain pending section when empty: %q", out)
	}
}

func TestFormatResumePrompt_IncludesPendingTasks(t *testing.T) {
	state := domain.SessionState{
		RecentTurns:  []domain.Turn{{Role: "user", Content: "hi"}},
		PendingTasks: []string{"Write tests", "Update docs"},
	}
	out := formatResumePrompt(state)
	if !strings.Contains(out, "- [ ] Write tests") {
		t.Errorf("missing pending task: %q", out)
	}
	if !strings.Contains(out, "- [ ] Update docs") {
		t.Errorf("missing pending task: %q", out)
	}
}

func TestFormatResumePrompt_IncludesActiveFiles(t *testing.T) {
	state := domain.SessionState{
		RecentTurns: []domain.Turn{{Role: "user", Content: "hi"}},
		ActiveFiles: []string{"main.go", "handler.go"},
	}
	out := formatResumePrompt(state)
	if !strings.Contains(out, "- main.go") {
		t.Errorf("missing active file: %q", out)
	}
}

func TestFormatResumePrompt_IncludesGitVision(t *testing.T) {
	state := domain.SessionState{
		RecentTurns: []domain.Turn{{Role: "user", Content: "hi"}},
		GitVision:   []domain.FileMetadata{{Path: "main.go", Status: "M"}},
	}
	out := formatResumePrompt(state)
	if !strings.Contains(out, "- main.go [M]") {
		t.Errorf("missing git vision entry: %q", out)
	}
}

func TestFormatResumePrompt_IncludesDiff(t *testing.T) {
	state := domain.SessionState{
		RecentTurns: []domain.Turn{{Role: "user", Content: "hi"}},
		Diff:        "+added line",
	}
	out := formatResumePrompt(state)
	if !strings.Contains(out, "```diff") || !strings.Contains(out, "+added line") {
		t.Errorf("missing diff block: %q", out)
	}
}

func TestFormatResumePrompt_OmitsDiffWhenEmpty(t *testing.T) {
	state := domain.SessionState{
		RecentTurns: []domain.Turn{{Role: "user", Content: "hi"}},
	}
	out := formatResumePrompt(state)
	if strings.Contains(out, "```diff") {
		t.Errorf("should not contain diff block when empty: %q", out)
	}
}

// ---- filterIgnored / filterIgnoredStrings ----

func makeIgnoreRules(t *testing.T, patterns string) *ignore.Rules {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ignore.IgnoreFileName)
	if err := os.WriteFile(path, []byte(patterns), 0644); err != nil {
		t.Fatal(err)
	}
	rules, err := ignore.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return rules
}

func TestFilterIgnored_RemovesMatchedFiles(t *testing.T) {
	rules := makeIgnoreRules(t, "vendor/\n")
	files := []domain.FileMetadata{
		{Path: "main.go"},
		{Path: "vendor/lib.go"},
	}
	got := filterIgnored(files, func(f domain.FileMetadata) string { return f.Path }, rules)
	if len(got) != 1 || got[0].Path != "main.go" {
		t.Errorf("expected only main.go, got %v", got)
	}
}

func TestFilterIgnored_EmptyRulesKeepsAll(t *testing.T) {
	rules := makeIgnoreRules(t, "")
	files := []domain.FileMetadata{{Path: "a.go"}, {Path: "b.go"}}
	got := filterIgnored(files, func(f domain.FileMetadata) string { return f.Path }, rules)
	if len(got) != 2 {
		t.Errorf("expected 2 files, got %d", len(got))
	}
}

func TestFilterIgnoredStrings_RemovesMatched(t *testing.T) {
	rules := makeIgnoreRules(t, "*.log\n")
	paths := []string{"main.go", "debug.log", "handler.go"}
	got := filterIgnoredStrings(paths, rules)
	for _, p := range got {
		if strings.HasSuffix(p, ".log") {
			t.Errorf("log file should be filtered: %q", p)
		}
	}
	if len(got) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(got), got)
	}
}

// ---- filterExistingPaths ----

func TestFilterExistingPaths_KeepsExistingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	got := filterExistingPaths(dir, []string{"main.go", "missing.go"})
	if len(got) != 1 || got[0] != "main.go" {
		t.Errorf("expected [main.go], got %v", got)
	}
}

func TestFilterExistingPaths_ExcludesDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "internal"), 0755); err != nil {
		t.Fatal(err)
	}
	got := filterExistingPaths(dir, []string{"internal"})
	if len(got) != 0 {
		t.Errorf("directories should be excluded, got %v", got)
	}
}

func TestFilterExistingPaths_HandlesForwardSlashOnWindows(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "util.go"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	// path stored with forward slash (as in YAML)
	got := filterExistingPaths(dir, []string{"pkg/util.go"})
	if len(got) != 1 {
		t.Errorf("expected forward-slash path to resolve, got %v", got)
	}
}
