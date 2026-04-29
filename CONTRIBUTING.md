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

Requirements: Go 1.21+

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
│   └── gemini_adapter.go ← Example: parses ~/.gemini/tmp/
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
        NewMyCLIAdapter(), // ← add here
    )
}
```

> **Order matters.** `DetectActiveAdapter` returns the first match. Put more specific adapters (project-local history files) before global ones.

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
| History file is JSONL | Use `bufio.Scanner` with a 1MB buffer (`scanner.Buffer(make([]byte, 1<<20), 1<<20)`) |
| History file is Markdown | Parse role headers line-by-line like `aider_adapter.go` |
| Tool stores history globally (not per-project) | Note it in a comment on `Detect`; still safe to add |
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
