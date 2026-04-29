package domain

import (
	"testing"
)

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
