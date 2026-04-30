package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAiderHistory(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, ".aider.chat.history.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAiderDetect(t *testing.T) {
	dir := t.TempDir()
	adapter := NewAiderAdapter()

	if adapter.Detect(dir) {
		t.Fatal("expected false when history file is absent")
	}

	writeAiderHistory(t, dir, "")
	if !adapter.Detect(dir) {
		t.Fatal("expected true when history file exists")
	}
}

func TestAiderExtractBasicTurns(t *testing.T) {
	dir := t.TempDir()
	writeAiderHistory(t, dir, `# USER
Hello, fix this bug.

# ASSISTANT
Sure, I'll fix it.
`)
	state, err := NewAiderAdapter().Extract(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.RecentTurns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(state.RecentTurns))
	}
	if state.RecentTurns[0].Role != "user" {
		t.Errorf("expected role user, got %q", state.RecentTurns[0].Role)
	}
	if !strings.Contains(state.RecentTurns[0].Content, "Hello, fix this bug.") {
		t.Errorf("unexpected user content: %q", state.RecentTurns[0].Content)
	}
	if state.RecentTurns[1].Role != "assistant" {
		t.Errorf("expected role assistant, got %q", state.RecentTurns[1].Role)
	}
}

func TestAiderExtractCommandPrefix(t *testing.T) {
	// "#### /<command>" lines should be treated as user turns.
	dir := t.TempDir()
	writeAiderHistory(t, dir, `#### /add main.go

# ASSISTANT
Added main.go to the chat.
`)
	state, err := NewAiderAdapter().Extract(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.RecentTurns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(state.RecentTurns))
	}
	if state.RecentTurns[0].Role != "user" {
		t.Errorf("expected user role for command line, got %q", state.RecentTurns[0].Role)
	}
}

func TestAiderExtractQuoteFormat(t *testing.T) {
	// "> USER" / "> ASSISTANT" is an alternative header format used by some versions.
	dir := t.TempDir()
	writeAiderHistory(t, dir, `> USER
Refactor this function.

> ASSISTANT
Done.
`)
	state, err := NewAiderAdapter().Extract(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.RecentTurns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(state.RecentTurns))
	}
	if state.RecentTurns[0].Role != "user" {
		t.Errorf("expected user, got %q", state.RecentTurns[0].Role)
	}
	if state.RecentTurns[1].Role != "assistant" {
		t.Errorf("expected assistant, got %q", state.RecentTurns[1].Role)
	}
}

func TestAiderExtractCodeBlockTruncation(t *testing.T) {
	// Code blocks longer than 50 lines must be truncated.
	var sb strings.Builder
	sb.WriteString("# USER\nShow me a big file.\n\n# ASSISTANT\n")
	sb.WriteString("Here:\n")
	sb.WriteString("```go\n")
	for range 80 {
		sb.WriteString("line\n")
	}
	sb.WriteString("```\n")

	dir := t.TempDir()
	writeAiderHistory(t, dir, sb.String())

	state, err := NewAiderAdapter().Extract(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assistantContent := state.RecentTurns[1].Content
	if !strings.Contains(assistantContent, "[Code block truncated for brevity]") {
		t.Errorf("expected truncation marker, got: %q", assistantContent)
	}
	// Should NOT contain line 80 of the original block.
	lineCount := strings.Count(assistantContent, "line\n")
	if lineCount > 50 {
		t.Errorf("expected at most 50 code lines, got %d", lineCount)
	}
}

func TestAiderExtractShortCodeBlockNotTruncated(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("# USER\nShow snippet.\n\n# ASSISTANT\n")
	sb.WriteString("```go\n")
	for range 10 {
		sb.WriteString("line\n")
	}
	sb.WriteString("```\n")

	dir := t.TempDir()
	writeAiderHistory(t, dir, sb.String())

	state, err := NewAiderAdapter().Extract(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assistantContent := state.RecentTurns[1].Content
	if strings.Contains(assistantContent, "[Code block truncated for brevity]") {
		t.Errorf("short code block should not be truncated")
	}
	if strings.Count(assistantContent, "line\n") != 10 {
		t.Errorf("expected 10 code lines intact")
	}
}

func TestAiderExtractRawBytesPopulated(t *testing.T) {
	content := "# USER\nHi.\n\n# ASSISTANT\nHello.\n"
	dir := t.TempDir()
	writeAiderHistory(t, dir, content)

	state, err := NewAiderAdapter().Extract(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.RawBytes != int64(len(content)) {
		t.Errorf("expected RawBytes=%d, got %d", len(content), state.RawBytes)
	}
}

func TestAiderExtractMissingFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	_, err := NewAiderAdapter().Extract(dir)
	if err == nil {
		t.Fatal("expected error when history file is missing")
	}
}
