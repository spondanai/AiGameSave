package repository

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func initGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runGitForTest(t, dir, "init")
	runGitForTest(t, dir, "config", "user.email", "test@example.com")
	runGitForTest(t, dir, "config", "user.name", "Test User")

	path := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(path, []byte("initial\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitForTest(t, dir, "add", "tracked.txt")
	runGitForTest(t, dir, "commit", "-m", "initial")
	return dir
}

func runGitForTest(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func TestGetDiffCapsTrackedDiffWithoutReadingAllIntoResult(t *testing.T) {
	dir := initGitRepo(t)

	var b strings.Builder
	for i := 0; i < 2000; i++ {
		b.WriteString("changed line\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	diff := GetDiff(dir, 256)
	if !strings.Contains(diff, "... [diff truncated] ...") {
		t.Fatalf("expected diff truncation marker, got:\n%s", diff)
	}
	if len(diff) > 256+len("\n... [diff truncated] ...") {
		t.Fatalf("diff exceeded cap plus marker: got %d bytes", len(diff))
	}
}

func TestGetDiffIncludesSmallUntrackedTextFile(t *testing.T) {
	dir := initGitRepo(t)

	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("remember this\n"), 0644); err != nil {
		t.Fatal(err)
	}

	diff := GetDiff(dir, 1024)
	if !strings.Contains(diff, "--- untracked: notes.txt ---") {
		t.Fatalf("expected untracked header, got:\n%s", diff)
	}
	if !strings.Contains(diff, "remember this") {
		t.Fatalf("expected untracked content, got:\n%s", diff)
	}
}

func TestGetDiffSkipsLargeUntrackedFile(t *testing.T) {
	dir := initGitRepo(t)

	large := strings.Repeat("x", 1024*1024+1)
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), []byte(large), 0644); err != nil {
		t.Fatal(err)
	}

	diff := GetDiff(dir, 4096)
	if !strings.Contains(diff, "--- untracked: large.txt [skipped: too large] ---") {
		t.Fatalf("expected large-file skip marker, got:\n%s", diff)
	}
	if strings.Contains(diff, strings.Repeat("x", 128)) {
		t.Fatalf("large untracked content should not be included")
	}
}

func TestGetDiffSkipsBinaryUntrackedFile(t *testing.T) {
	dir := initGitRepo(t)

	if err := os.WriteFile(filepath.Join(dir, "blob.bin"), []byte{'a', 0, 'b'}, 0644); err != nil {
		t.Fatal(err)
	}

	diff := GetDiff(dir, 1024)
	if !strings.Contains(diff, "--- untracked: blob.bin ---") {
		t.Fatalf("expected binary untracked header, got:\n%s", diff)
	}
	if !strings.Contains(diff, "[skipped: binary file]") {
		t.Fatalf("expected binary skip marker, got:\n%s", diff)
	}
}
