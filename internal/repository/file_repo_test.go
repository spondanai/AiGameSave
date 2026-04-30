package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spondanai/aigamesave/internal/domain"
)

func buildState() domain.SessionState {
	return domain.SessionState{
		RecentTurns: []domain.Turn{
			{Role: "user", Content: "Fix the bug"},
			{Role: "assistant", Content: "Done."},
		},
		ActiveFiles:  []string{"main.go", "handler.go"},
		PendingTasks: []string{"Write tests"},
		GitVision: []domain.FileMetadata{
			{Path: "main.go", Status: "M"},
		},
		Diff: "diff --git a/main.go b/main.go\n+fixed",
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := buildState()

	if err := SaveSession(dir, want); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	got, err := LoadSession(dir)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}

	if len(got.RecentTurns) != len(want.RecentTurns) {
		t.Fatalf("turns: want %d got %d", len(want.RecentTurns), len(got.RecentTurns))
	}
	if got.RecentTurns[0].Content != want.RecentTurns[0].Content {
		t.Errorf("turn[0] content mismatch: %q", got.RecentTurns[0].Content)
	}
	if len(got.ActiveFiles) != len(want.ActiveFiles) {
		t.Errorf("active files: want %d got %d", len(want.ActiveFiles), len(got.ActiveFiles))
	}
	if len(got.PendingTasks) != 1 || got.PendingTasks[0] != "Write tests" {
		t.Errorf("pending tasks mismatch: %v", got.PendingTasks)
	}
	if got.Diff != want.Diff {
		t.Errorf("diff mismatch: %q", got.Diff)
	}
}

func TestSaveCreatesFile(t *testing.T) {
	dir := t.TempDir()
	if err := SaveSession(dir, buildState()); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, SaveFileName)); err != nil {
		t.Errorf("save file not created: %v", err)
	}
}

func TestSaveFilePermissions(t *testing.T) {
	dir := t.TempDir()
	if err := SaveSession(dir, buildState()); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, SaveFileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected mode 0600, got %o", info.Mode().Perm())
	}
}

func TestRawBytesNotPersisted(t *testing.T) {
	dir := t.TempDir()
	state := buildState()
	state.RawBytes = 99999
	if err := SaveSession(dir, state); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	got, err := LoadSession(dir)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if got.RawBytes != 0 {
		t.Errorf("RawBytes should not persist, got %d", got.RawBytes)
	}
}

func TestLoadSessionMissingFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadSession(dir)
	if err == nil {
		t.Fatal("expected error when save file is absent")
	}
}

func TestEnsureGitIgnore_CreatesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	if err := SaveSession(dir, buildState()); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf(".gitignore not created: %v", err)
	}
	if !strings.Contains(string(data), SaveFileName) {
		t.Errorf(".gitignore does not contain %q: %s", SaveFileName, data)
	}
}

func TestEnsureGitIgnore_DoesNotDuplicate(t *testing.T) {
	dir := t.TempDir()
	// Write a .gitignore that already contains the save file name.
	existing := "*.log\n" + SaveFileName + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SaveSession(dir, buildState()); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), SaveFileName) != 1 {
		t.Errorf("expected exactly one occurrence of %q, got:\n%s", SaveFileName, data)
	}
}

func TestEnsureGitIgnore_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	// .gitignore exists but does not mention the save file; no trailing newline.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SaveSession(dir, buildState()); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "*.log") {
		t.Errorf("original content lost: %s", data)
	}
	if !strings.Contains(string(data), SaveFileName) {
		t.Errorf("%q not appended: %s", SaveFileName, data)
	}
}
