package adapters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// buildClaudeSession writes a synthetic Claude Code JSONL session and returns
// a ClaudeAdapter wired to use it as the "latest" session.
func buildClaudeSession(t *testing.T, lines []map[string]any) *ClaudeAdapter {
	t.Helper()

	homeDir := t.TempDir()
	projectDir := filepath.Join(homeDir, ".claude", "projects", "-tmp-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(projectDir, "session.jsonl")
	f, err := os.Create(sessionPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, line := range lines {
		if err := enc.Encode(line); err != nil {
			t.Fatal(err)
		}
	}

	// Monkey-patch: return an adapter that hardcodes the session path.
	// We do this by embedding a fake findClaudeProjectDir result via a
	// test-only helper that directly calls the extraction logic.
	return &ClaudeAdapter{projectDirOverride: projectDir}
}

func claudeUserMsg(text string) map[string]any {
	return map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": text,
		},
	}
}

func claudeAssistantMsg(blocks ...map[string]any) map[string]any {
	return map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"role":    "assistant",
			"content": blocks,
		},
	}
}

func claudeTextBlock(text string) map[string]any {
	return map[string]any{"type": "text", "text": text}
}

func claudeTodoWriteBlock(todos []map[string]any) map[string]any {
	return map[string]any{
		"type":  "tool_use",
		"name":  "TodoWrite",
		"input": map[string]any{"todos": todos},
	}
}

func TestClaudeExtractPendingTasks(t *testing.T) {
	todos := []map[string]any{
		{"content": "Write unit tests", "status": "pending", "priority": "high"},
		{"content": "Update README", "status": "in_progress", "priority": "medium"},
		{"content": "Fix linter", "status": "completed", "priority": "low"},
	}

	adapter := buildClaudeSession(t, []map[string]any{
		claudeUserMsg("Can you write tests and update the docs?"),
		claudeAssistantMsg(
			claudeTextBlock("Sure, I'll start with the tests."),
			claudeTodoWriteBlock(todos),
		),
	})

	state, err := adapter.extractFromPath(adapter.projectDirOverride)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.PendingTasks) != 2 {
		t.Fatalf("expected 2 pending tasks, got %d: %v", len(state.PendingTasks), state.PendingTasks)
	}
	if state.PendingTasks[0] != "Write unit tests" {
		t.Errorf("unexpected task[0]: %q", state.PendingTasks[0])
	}
	if state.PendingTasks[1] != "Update README" {
		t.Errorf("unexpected task[1]: %q", state.PendingTasks[1])
	}
}

func TestClaudeExtractUsesLastTodoWrite(t *testing.T) {
	firstTodos := []map[string]any{
		{"content": "Old task", "status": "pending", "priority": "high"},
	}
	secondTodos := []map[string]any{
		{"content": "New task", "status": "pending", "priority": "high"},
		{"content": "Another new task", "status": "in_progress", "priority": "medium"},
	}

	adapter := buildClaudeSession(t, []map[string]any{
		claudeUserMsg("Step 1"),
		claudeAssistantMsg(claudeTextBlock("Starting."), claudeTodoWriteBlock(firstTodos)),
		claudeUserMsg("Step 2"),
		claudeAssistantMsg(claudeTextBlock("Updated plan."), claudeTodoWriteBlock(secondTodos)),
	})

	state, err := adapter.extractFromPath(adapter.projectDirOverride)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.PendingTasks) != 2 {
		t.Fatalf("expected 2 tasks from last TodoWrite, got %d", len(state.PendingTasks))
	}
	if state.PendingTasks[0] != "New task" {
		t.Errorf("expected latest TodoWrite tasks, got: %v", state.PendingTasks)
	}
}

func TestClaudeExtractNoPendingTasksWhenAllCompleted(t *testing.T) {
	todos := []map[string]any{
		{"content": "Done thing", "status": "completed", "priority": "high"},
	}

	adapter := buildClaudeSession(t, []map[string]any{
		claudeUserMsg("All done?"),
		claudeAssistantMsg(claudeTextBlock("Yes!"), claudeTodoWriteBlock(todos)),
	})

	state, err := adapter.extractFromPath(adapter.projectDirOverride)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.PendingTasks) != 0 {
		t.Errorf("expected no pending tasks, got %v", state.PendingTasks)
	}
}

func TestClaudeExtractNoTodoWrite(t *testing.T) {
	adapter := buildClaudeSession(t, []map[string]any{
		claudeUserMsg("Hello"),
		claudeAssistantMsg(claudeTextBlock("Hi there!")),
	})

	state, err := adapter.extractFromPath(adapter.projectDirOverride)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.PendingTasks) != 0 {
		t.Errorf("expected no pending tasks, got %v", state.PendingTasks)
	}
}
