package domain

import (
	"testing"
	"time"
)

func TestRankFiles_MentionBonusWins(t *testing.T) {
	now := time.Now()
	files := []FileMetadata{
		{Path: "old.go", ModTime: now.Add(-10 * time.Hour)},
		{Path: "hot.go", ModTime: now.Add(-1 * time.Hour)},
	}
	turns := []Turn{
		{Role: "user", Content: "please fix old.go"},
	}
	ranked := RankFiles(files, turns)
	if ranked[0].Path != "old.go" {
		t.Errorf("mentioned file should rank first, got %q", ranked[0].Path)
	}
}

func TestRankFiles_MtimeBreaksTie(t *testing.T) {
	now := time.Now()
	files := []FileMetadata{
		{Path: "a.go", ModTime: now.Add(-2 * time.Hour)},
		{Path: "b.go", ModTime: now.Add(-1 * time.Hour)},
	}
	ranked := RankFiles(files, nil)
	if ranked[0].Path != "b.go" {
		t.Errorf("most recently modified file should rank first, got %q", ranked[0].Path)
	}
}

func TestRankFiles_CapsAtMaxFiles(t *testing.T) {
	now := time.Now()
	files := make([]FileMetadata, maxFiles+5)
	for i := range files {
		files[i] = FileMetadata{Path: "file.go", ModTime: now}
	}
	ranked := RankFiles(files, nil)
	if len(ranked) != maxFiles {
		t.Errorf("expected %d files, got %d", maxFiles, len(ranked))
	}
}

func TestRankFiles_EmptyInputReturnsEmpty(t *testing.T) {
	if ranked := RankFiles(nil, nil); len(ranked) != 0 {
		t.Errorf("expected empty result, got %v", ranked)
	}
}

func TestRankFiles_SetsPriority(t *testing.T) {
	files := []FileMetadata{{Path: "main.go", ModTime: time.Now()}}
	ranked := RankFiles(files, nil)
	if ranked[0].Priority == 0 {
		t.Errorf("expected non-zero priority for most recent file")
	}
}

func TestRankFiles_MentionBonusDoesNotStack(t *testing.T) {
	now := time.Now()
	files := []FileMetadata{{Path: "main.go", ModTime: now}}
	turns := []Turn{
		{Role: "user", Content: "fix main.go"},
		{Role: "assistant", Content: "looking at main.go now"},
	}
	single := RankFiles([]FileMetadata{{Path: "main.go", ModTime: now}}, turns[:1])
	multi := RankFiles(files, turns)
	if single[0].Priority != multi[0].Priority {
		t.Errorf("mention bonus should not stack: single=%d multi=%d",
			single[0].Priority, multi[0].Priority)
	}
}

func TestExtractFilePaths_BasicPaths(t *testing.T) {
	turns := []Turn{
		{Role: "assistant", Content: "สร้าง internal/adapters/context.go แล้ว"},
		{Role: "assistant", Content: "แก้ internal/adapters/registry.go และ cmd/ags/main.go ด้วย"},
	}
	paths := ExtractFilePaths(turns)

	want := map[string]bool{
		"internal/adapters/context.go":  true,
		"internal/adapters/registry.go": true,
		"cmd/ags/main.go":               true,
	}
	for _, p := range paths {
		if !want[p] {
			t.Errorf("unexpected path: %q", p)
		}
		delete(want, p)
	}
	for missing := range want {
		t.Errorf("missing path: %q", missing)
	}
}

func TestExtractFilePaths_SkipsURLs(t *testing.T) {
	turns := []Turn{
		{Role: "user", Content: "ดู https://github.com/spondanai/aigamesave/cmd/ags/main.go"},
		{Role: "assistant", Content: "go install github.com/spondanai/aigamesave/cmd/ags@latest"},
	}
	paths := ExtractFilePaths(turns)
	for _, p := range paths {
		t.Errorf("expected no paths, got: %q", p)
	}
}

func TestExtractFilePaths_Deduplication(t *testing.T) {
	turns := []Turn{
		{Role: "user", Content: "แก้ internal/adapters/context.go"},
		{Role: "assistant", Content: "internal/adapters/context.go อัปเดตแล้ว"},
	}
	paths := ExtractFilePaths(turns)
	if len(paths) != 1 {
		t.Errorf("expected 1 unique path, got %d: %v", len(paths), paths)
	}
}

func TestExtractFilePaths_SkipsVersionStrings(t *testing.T) {
	turns := []Turn{
		{Role: "assistant", Content: "ใช้ version 1.2/3.4.5 และ go1.21/1.22.0"},
	}
	// These start with digits so the regex should not match
	paths := ExtractFilePaths(turns)
	for _, p := range paths {
		t.Errorf("unexpected version string captured: %q", p)
	}
}

func TestExtractFilePaths_WindowsBackslash(t *testing.T) {
	turns := []Turn{
		{Role: "assistant", Content: `แก้ internal\adapters\registry.go และ cmd\ags\main.go แล้ว`},
	}
	paths := ExtractFilePaths(turns)

	want := map[string]bool{
		"internal/adapters/registry.go": true,
		"cmd/ags/main.go":               true,
	}
	for _, p := range paths {
		if !want[p] {
			t.Errorf("unexpected path: %q", p)
		}
		delete(want, p)
	}
	for missing := range want {
		t.Errorf("missing path: %q", missing)
	}
}

func TestExtractFilePaths_NormalizesBackslashForDedup(t *testing.T) {
	// Same file mentioned once with / and once with \  — should deduplicate to one entry.
	turns := []Turn{
		{Role: "user", Content: `internal/adapters/context.go`},
		{Role: "assistant", Content: `internal\adapters\context.go อัปเดตแล้ว`},
	}
	paths := ExtractFilePaths(turns)
	if len(paths) != 1 {
		t.Errorf("expected 1 deduplicated path, got %d: %v", len(paths), paths)
	}
	if len(paths) == 1 && paths[0] != "internal/adapters/context.go" {
		t.Errorf("expected forward-slash path, got %q", paths[0])
	}
}

func TestLooksLikeDomain(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"github.com/foo/bar.go", true},
		{"pkg.go.dev/foo/bar", true},
		{"internal/adapters/context.go", false},
		{"cmd/ags/main.go", false},
		{`github.com\foo\bar.go`, true},   // backslash variant
		{`internal\adapters\context.go`, false},
	}
	for _, c := range cases {
		if got := looksLikeDomain(c.path); got != c.want {
			t.Errorf("looksLikeDomain(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
