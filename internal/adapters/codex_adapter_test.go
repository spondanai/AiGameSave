package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeCodexSession(t *testing.T, codexHome, workingDir, name string, lines ...string) string {
	t.Helper()

	sessionDir := filepath.Join(codexHome, "sessions", "2026", "04", "29")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(sessionDir, name)
	allLines := []string{
		`{"type":"session_meta","payload":{"cwd":` + quoteJSON(workingDir) + `}}`,
	}
	allLines = append(allLines, lines...)
	if err := os.WriteFile(path, []byte(strings.Join(allLines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func quoteJSON(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func TestCodexAdapterDetectsSessionForWorkingDir(t *testing.T) {
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	workingDir := t.TempDir()

	writeCodexSession(t, codexHome, workingDir, "rollout-test.jsonl")

	adapter := NewCodexAdapter()
	if !adapter.Detect(workingDir) {
		t.Fatal("expected Codex session to be detected")
	}
}

func TestCodexAdapterExtractsRecentUserAndAssistantMessages(t *testing.T) {
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	workingDir := t.TempDir()

	writeCodexSession(t, codexHome, workingDir, "rollout-test.jsonl",
		`{"type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"skip me"}]}}`,
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"please continue"}]}}`,
		`{"type":"response_item","payload":{"type":"function_call","name":"exec_command"}}`,
		`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"working on it"}]}}`,
	)

	state, err := NewCodexAdapter().Extract(workingDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.RecentTurns) != 2 {
		t.Fatalf("expected 2 turns, got %d: %#v", len(state.RecentTurns), state.RecentTurns)
	}
	if state.RecentTurns[0].Role != "user" || state.RecentTurns[0].Content != "please continue" {
		t.Fatalf("unexpected first turn: %#v", state.RecentTurns[0])
	}
	if state.RecentTurns[1].Role != "assistant" || state.RecentTurns[1].Content != "working on it" {
		t.Fatalf("unexpected second turn: %#v", state.RecentTurns[1])
	}
}

func TestCodexAdapterUsesLatestMatchingSession(t *testing.T) {
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	workingDir := t.TempDir()

	oldPath := writeCodexSession(t, codexHome, workingDir, "rollout-old.jsonl",
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"old"}]}}`,
	)
	newPath := writeCodexSession(t, codexHome, workingDir, "rollout-new.jsonl",
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"new"}]}}`,
	)

	oldTime := time.Now().Add(-time.Hour)
	newTime := time.Now()
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	state, err := NewCodexAdapter().Extract(workingDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.RecentTurns) != 1 || state.RecentTurns[0].Content != "new" {
		t.Fatalf("expected latest matching session, got %#v", state.RecentTurns)
	}
}

func TestCodexAdapterIgnoresDifferentWorkingDir(t *testing.T) {
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	workingDir := t.TempDir()
	otherDir := t.TempDir()

	writeCodexSession(t, codexHome, otherDir, "rollout-other.jsonl",
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"wrong project"}]}}`,
	)

	if NewCodexAdapter().Detect(workingDir) {
		t.Fatal("did not expect Codex session from another working dir to match")
	}
}
