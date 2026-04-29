package ignore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIgnored_DirectorySlash(t *testing.T) {
	r := rulesFrom("vendor/\ntestdata/\n")
	cases := map[string]bool{
		"vendor/pkg/foo.go":     true,
		"vendor":                true,
		"testdata/fixture.json": true,
		"internal/vendor/x.go": false,
		"cmd/main.go":           false,
	}
	checkCases(t, r, cases)
}

func TestIgnored_DirectoryDoublestar(t *testing.T) {
	r := rulesFrom("vendor/**\n")
	cases := map[string]bool{
		"vendor/pkg/foo.go": true,
		"vendor":            true,
		"cmd/main.go":       false,
	}
	checkCases(t, r, cases)
}

func TestIgnored_BasenameGlob(t *testing.T) {
	r := rulesFrom("*.pb.go\n*.gen.go\n")
	cases := map[string]bool{
		"internal/foo.pb.go":           true,
		"pkg/bar.gen.go":               true,
		"internal/adapters/context.go": false,
		"go.sum":                       false,
	}
	checkCases(t, r, cases)
}

func TestIgnored_DoublestarPrefix(t *testing.T) {
	r := rulesFrom("**/*.pb.go\n")
	cases := map[string]bool{
		"internal/foo.pb.go": true,
		"foo.pb.go":          true,
		"foo.go":             false,
	}
	checkCases(t, r, cases)
}

func TestIgnored_FullPathGlob(t *testing.T) {
	r := rulesFrom("internal/generated/*.go\n")
	cases := map[string]bool{
		"internal/generated/foo.go": true,
		"internal/adapters/foo.go":  false,
		"cmd/main.go":               false,
	}
	checkCases(t, r, cases)
}

func TestIgnored_CommentsAndBlanks(t *testing.T) {
	r := rulesFrom("# this is a comment\n\nvendor/\n# another comment\n")
	if !r.Ignored("vendor/pkg/foo.go") {
		t.Error("expected vendor/ to be ignored")
	}
	if r.Ignored("cmd/main.go") {
		t.Error("cmd/main.go should not be ignored")
	}
}

func TestLoad_MissingFileReturnsEmptyRules(t *testing.T) {
	dir := t.TempDir()
	rules, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error for missing file: %v", err)
	}
	if rules.Ignored("anything/file.go") {
		t.Error("empty rules should not ignore anything")
	}
}

func TestLoad_ParsesFile(t *testing.T) {
	dir := t.TempDir()
	content := "# exclude generated\n*.pb.go\nvendor/\n"
	if err := os.WriteFile(filepath.Join(dir, IgnoreFileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	rules, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !rules.Ignored("internal/foo.pb.go") {
		t.Error("*.pb.go pattern should match internal/foo.pb.go")
	}
	if !rules.Ignored("vendor/pkg/bar.go") {
		t.Error("vendor/ pattern should match vendor/pkg/bar.go")
	}
	if rules.Ignored("cmd/main.go") {
		t.Error("cmd/main.go should not be ignored")
	}
}

// rulesFrom builds a Rules value directly from a multiline string without
// touching the filesystem — convenient for table-driven tests.
func rulesFrom(content string) *Rules {
	r := &Rules{}
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		r.patterns = append(r.patterns, filepath.ToSlash(line))
	}
	return r
}

func checkCases(t *testing.T, r *Rules, cases map[string]bool) {
	t.Helper()
	for path, want := range cases {
		if got := r.Ignored(path); got != want {
			t.Errorf("Ignored(%q) = %v, want %v", path, got, want)
		}
	}
}
