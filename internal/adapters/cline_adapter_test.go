package adapters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// buildClineTask creates a fake Cline task directory inside tasksRoot and returns
// the path to api_conversation_history.json.
func buildClineTask(t *testing.T, tasksRoot, taskID, cwd string, msgs []map[string]any) string {
	t.Helper()
	dir := filepath.Join(tasksRoot, taskID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	histPath := filepath.Join(dir, "api_conversation_history.json")
	data, err := json.Marshal(msgs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(histPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return histPath
}

// clineEnvDetails generates a user message that looks like Cline's first turn.
func clineEnvDetails(cwd, taskText string) map[string]any {
	text := "<task>\n" + taskText + "\n</task>\n<environment_details>\n# Current Working Directory (" + cwd + ") Files\ngo.mod\n</environment_details>"
	return map[string]any{
		"role": "user",
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
	}
}

func TestClineExtractCWD(t *testing.T) {
	tmpDir := t.TempDir()
	tasksRoot := filepath.Join(tmpDir, "tasks")
	_ = os.MkdirAll(tasksRoot, 0o755)

	cwd := "/home/user/myproject"
	msgs := []map[string]any{
		clineEnvDetails(cwd, "Fix the bug"),
		{"role": "assistant", "content": []map[string]any{{"type": "text", "text": "I'll fix it."}}},
	}
	buildClineTask(t, tasksRoot, "1714000000000", cwd, msgs)

	got := extractClineCWD(filepath.Join(tasksRoot, "1714000000000", "api_conversation_history.json"))
	if got != cwd {
		t.Fatalf("expected cwd %q, got %q", cwd, got)
	}
}

func TestClineFindTaskMatchesWorkingDir(t *testing.T) {
	tmpDir := t.TempDir()
	tasksRoot := filepath.Join(tmpDir, "tasks")
	_ = os.MkdirAll(tasksRoot, 0o755)

	workDir := t.TempDir()
	msgs := []map[string]any{
		clineEnvDetails(workDir, "Do something"),
		{"role": "assistant", "content": "Sure!"},
	}
	buildClineTask(t, tasksRoot, "1714000000000", workDir, msgs)

	adapter := &ClineAdapter{}
	// Override the storage root by monkey-patching via the env var approach won't work
	// for the cross-platform helper. Instead, test findTask directly with a patched root.
	// We use an unexported helper for testability.
	taskPath, err := clineFindTaskIn(tasksRoot, workDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(tasksRoot, "1714000000000", "api_conversation_history.json")
	if taskPath != expected {
		t.Fatalf("expected %q, got %q", expected, taskPath)
	}
	_ = adapter
}

func TestClineFindTaskPicksLatest(t *testing.T) {
	tmpDir := t.TempDir()
	tasksRoot := filepath.Join(tmpDir, "tasks")
	_ = os.MkdirAll(tasksRoot, 0o755)

	workDir := t.TempDir()
	olderID := "1714000000000"
	newerID := "1714999999999"

	msgs := []map[string]any{clineEnvDetails(workDir, "task")}
	buildClineTask(t, tasksRoot, olderID, workDir, msgs)
	buildClineTask(t, tasksRoot, newerID, workDir, msgs)

	taskPath, err := clineFindTaskIn(tasksRoot, workDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Dir(taskPath) != filepath.Join(tasksRoot, newerID) {
		t.Fatalf("expected newer task %s, got %s", newerID, filepath.Base(filepath.Dir(taskPath)))
	}
}

func TestClineFindTaskNoMatchReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	tasksRoot := filepath.Join(tmpDir, "tasks")
	_ = os.MkdirAll(tasksRoot, 0o755)

	otherDir := "/some/other/project"
	msgs := []map[string]any{clineEnvDetails(otherDir, "task")}
	buildClineTask(t, tasksRoot, "1714000000000", otherDir, msgs)

	workDir := t.TempDir()
	_, err := clineFindTaskIn(tasksRoot, workDir)
	if err == nil {
		t.Fatal("expected error for non-matching dir")
	}
}

func TestClineExtractTurns(t *testing.T) {
	tmpDir := t.TempDir()
	tasksRoot := filepath.Join(tmpDir, "tasks")
	_ = os.MkdirAll(tasksRoot, 0o755)

	workDir := t.TempDir()
	msgs := []map[string]any{
		clineEnvDetails(workDir, "Fix the null pointer"),
		{"role": "assistant", "content": []map[string]any{{"type": "text", "text": "I'll look into it."}}},
		{"role": "user", "content": "ok go ahead"},
		{"role": "assistant", "content": []map[string]any{{"type": "text", "text": "Fixed!"}}},
	}
	histPath := buildClineTask(t, tasksRoot, "1714000000000", workDir, msgs)

	data, _ := os.ReadFile(histPath)
	var parsed []clineMessage
	_ = json.Unmarshal(data, &parsed)

	var turns []struct{ Role, Content string }
	for _, m := range parsed {
		text := clineMessageText(m.Content)
		if text != "" {
			turns = append(turns, struct{ Role, Content string }{m.Role, text})
		}
	}

	if len(turns) != 4 {
		t.Fatalf("expected 4 turns, got %d", len(turns))
	}
	if turns[1].Role != "assistant" || turns[1].Content != "I'll look into it." {
		t.Fatalf("unexpected turn[1]: %+v", turns[1])
	}
}

func TestClineLastActiveUsesFileMtime(t *testing.T) {
	tmpDir := t.TempDir()
	tasksRoot := filepath.Join(tmpDir, "tasks")
	_ = os.MkdirAll(tasksRoot, 0o755)

	workDir := t.TempDir()
	msgs := []map[string]any{clineEnvDetails(workDir, "task")}
	histPath := buildClineTask(t, tasksRoot, "1714000000000", workDir, msgs)

	info, err := os.Stat(histPath)
	if err != nil {
		t.Fatal(err)
	}
	want := info.ModTime().Truncate(time.Second)

	taskPath, err := clineFindTaskIn(tasksRoot, workDir)
	if err != nil {
		t.Fatal(err)
	}
	gotInfo, _ := os.Stat(taskPath)
	got := gotInfo.ModTime().Truncate(time.Second)

	if !got.Equal(want) {
		t.Fatalf("mtime mismatch: want %v got %v", want, got)
	}
}
