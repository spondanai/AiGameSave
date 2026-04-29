# Contributing to AiGameSave (AGS)

Thanks for your interest in contributing! The most impactful way to help is adding support for a new AI CLI — and it only takes **one file**.

---

## Quick Start

```bash
git clone https://github.com/spondanai/aigamesave.git
cd aigamesave
go build ./...
go run ./cmd/ags save   # test against your own project
```

Requirements: the Go version declared in `go.mod`.

---

## Project Structure

```
internal/
├── domain/
│   ├── extractor.go      ← HistoryExtractor interface + core types
│   └── heuristic.go      ← Git file ranking logic
├── adapters/
│   ├── registry.go       ← Register your adapter here (one line)
│   ├── aider_adapter.go  ← Example: parses .aider.chat.history.md
│   ├── claude_adapter.go ← Example: parses ~/.claude/projects/
│   ├── gemini_adapter.go ← Example: parses ~/.gemini/tmp/
│   └── codex_adapter.go  ← Example: parses ~/.codex/sessions/
├── usecase/
│   └── gamesave_usecase.go ← Orchestration (no need to touch)
└── repository/
    ├── git_repo.go       ← git status wrapper
    └── file_repo.go      ← YAML save/load
```

The architecture is intentionally simple: **adapters read history → usecase combines with git status → repository saves YAML.**

---

## Adding a New AI CLI Adapter

### Step 1 — Create your adapter file

Create `internal/adapters/<name>_adapter.go` and implement the `HistoryExtractor` interface:

```go
package adapters

import "github.com/spondanai/aigamesave/internal/domain"

type MyCLIAdapter struct{}

func NewMyCLIAdapter() *MyCLIAdapter { return &MyCLIAdapter{} }

// Detect returns true if this AI CLI is active in workingDir.
// Check for a config file, history file, or directory specific to your tool.
func (a *MyCLIAdapter) Detect(workingDir string) bool {
    // Example: check for a history file in the project directory
    _, err := os.Stat(filepath.Join(workingDir, ".mycli_history"))
    return err == nil
}

// Extract reads the history and returns the last few conversation turns.
// Keep it to ~3 pairs (6 turns). Truncate long content to stay token-light.
func (a *MyCLIAdapter) Extract(workingDir string) (domain.SessionState, error) {
    // 1. Read your tool's history file
    // 2. Parse into []domain.Turn{Role: "user"/"assistant", Content: "..."}
    // 3. Truncate content longer than 1500 chars
    // 4. Return the last 6 turns
    return domain.SessionState{RecentTurns: turns}, nil
}

// LastActive returns the modification time of the matched session/history file.
// AGS uses this to choose the right adapter when multiple AI CLIs are present.
func (a *MyCLIAdapter) LastActive(workingDir string) (time.Time, error) {
    info, err := os.Stat(filepath.Join(workingDir, ".mycli_history"))
    if err != nil {
        return time.Time{}, err
    }
    return info.ModTime(), nil
}

func (a *MyCLIAdapter) Name() string { return "MyCLI" }
```

### Step 2 — Register it

Open `internal/adapters/registry.go` and add your adapter to `init()`:

```go
func init() {
    registry = append(registry,
        NewAiderAdapter(),
        NewClaudeAdapter(),
        NewGeminiAdapter(),
        NewCodexAdapter(),
        NewMyCLIAdapter(), // ← add here
    )
}
```

`DetectActiveAdapter` chooses the detected adapter with the newest `LastActive`
timestamp. This keeps `ags save` pointed at the AI CLI the user touched most
recently when several tools have sessions for the same project.

### Step 3 — Test it

```bash
cd /path/to/project/that/uses/mycli
go run /path/to/aigamesave/cmd/ags/main.go save
# Should print: Detected AI CLI: MyCLI
```

### Step 4 — Open a PR

Title format: `feat: add <ToolName> adapter`

Include in your PR description:
- Where the tool stores its history (path + format)
- A sample of the raw history file (redact any sensitive content)
- Screenshot or output of `ags save` working

---

## Interface Reference

```go
// From internal/domain/extractor.go

type HistoryExtractor interface {
    Detect(workingDir string) bool
    Extract(workingDir string) (SessionState, error)
    Name() string
}

// Adapters should also implement:
type activeAdapter interface {
    LastActive(workingDir string) (time.Time, error)
}

type SessionState struct {
    RecentTurns []Turn
    GitVision   []FileMetadata // populated by usecase, not your adapter
}

type Turn struct {
    Role    string // "user" or "assistant"
    Content string
}
```

---

## Adapter Tips

| Situation | Recommendation |
|---|---|
| History file is JSONL | Read line-by-line with `bufio.Reader.ReadLine` or a similar limited reader, and skip/truncate oversized lines |
| History file is Markdown | Parse role headers line-by-line like `aider_adapter.go` |
| Tool stores history globally (not per-project) | Correlate the session to `workingDir` using metadata, project hash, or cwd before `Detect` returns true |
| Content can be very long | Truncate at 1500 chars: `content[:1500] + "\n... [truncated] ..."` |
| Code blocks in history | Skip lines after 50 inside a code block (see `aider_adapter.go`) |

---

## Good First Issues

Check the [Issues tab](https://github.com/spondanai/aigamesave/issues) for adapters tagged **`good first issue`**:

- [ ] Cline (VS Code extension — stores history in `.cline/`)
- [ ] Cursor — chat history location TBD
- [ ] Copilot CLI

---

## Code Style

- No comments explaining *what* the code does — only *why* when non-obvious
- No error wrapping for internal errors that can't happen
- `gofmt` before committing (`go fmt ./...`)
- Keep adapters self-contained — don't add dependencies to `go.mod` unless absolutely necessary
