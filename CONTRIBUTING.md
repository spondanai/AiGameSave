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
│   ├── extractor.go        ← HistoryExtractor interface + core types
│   └── heuristic.go        ← Git file ranking logic
├── adapters/
│   ├── registry.go         ← Register your adapter here (one line)
│   ├── context.go          ← selectContext() — shared context selection logic
│   ├── aider_adapter.go    ← parses .aider.chat.history.md
│   ├── claude_adapter.go   ← parses ~/.claude/projects/
│   ├── gemini_adapter.go   ← parses ~/.gemini/tmp/
│   ├── codex_adapter.go    ← parses ~/.codex/sessions/
│   └── copilot_adapter.go  ← parses VS Code workspaceStorage/*/chatSessions/
├── usecase/
│   └── gamesave_usecase.go ← Orchestration (no need to touch)
└── repository/
    ├── git_repo.go         ← git status wrapper
    └── file_repo.go        ← YAML save/load
```

The architecture is intentionally simple: **adapters read history → usecase combines with git status → repository saves YAML.**

---

## Adding a New AI CLI Adapter

### Step 1 — Create your adapter file

Create `internal/adapters/<name>_adapter.go` and implement the `HistoryExtractor` interface:

```go
package adapters

import (
    "os"
    "path/filepath"
    "time"

    "github.com/spondanai/aigamesave/internal/domain"
)

type MyCLIAdapter struct{}

func NewMyCLIAdapter() *MyCLIAdapter { return &MyCLIAdapter{} }

// Detect returns true if this AI CLI is active in workingDir.
func (a *MyCLIAdapter) Detect(workingDir string) bool {
    _, err := os.Stat(filepath.Join(workingDir, ".mycli_history"))
    return err == nil
}

// Extract reads the history and returns conversation turns.
// Use selectContext() to select the anchor + tail — do NOT slice manually.
func (a *MyCLIAdapter) Extract(workingDir string) (domain.SessionState, error) {
    // 1. Read your tool's history file
    // 2. Parse into []domain.Turn{Role: "user"/"assistant", Content: "..."}
    // 3. Truncate individual content longer than 1500 chars
    // 4. Pass ALL collected turns to selectContext — it handles the rest
    return domain.SessionState{RecentTurns: selectContext(turns)}, nil
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
        NewCopilotAdapter(),
        NewMyCLIAdapter(), // ← add here
    )
}
```

`DetectActiveAdapter` picks the detected adapter with the newest `LastActive` timestamp, so `ags save` always targets the AI CLI the user touched most recently when several tools have sessions for the same project.

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
- Output of `ags save` working

---

## Interface Reference

```go
// From internal/domain/extractor.go

type HistoryExtractor interface {
    Detect(workingDir string) bool
    Extract(workingDir string) (SessionState, error)
    Name() string
}

// Adapters should also implement activeAdapter so AGS can pick the
// most recently active session when multiple CLIs coexist:
type activeAdapter interface {
    LastActive(workingDir string) (time.Time, error)
}

type SessionState struct {
    RecentTurns []Turn
    ActiveFiles []string       // populated by usecase (file-path extraction)
    GitVision   []FileMetadata // populated by usecase (git status)
    Diff        string         // populated by usecase (omitted if --no-diff)
}

type Turn struct {
    Role    string // "user" or "assistant"
    Content string
}
```

---

## Context Selection — `selectContext()`

Do **not** slice turns manually in your adapter. Always call `selectContext(turns)` from `internal/adapters/context.go`:

```go
return domain.SessionState{RecentTurns: selectContext(turns)}, nil
```

**What it does:** backward-searches for the last user turn with ≥10 characters (the "anchor"), then returns that anchor plus the most recent 5 turns that followed. Short acknowledgements like "ok", "โอเค", "ลองรันดู" are automatically skipped so the anchor stays on the user's actual instruction — not the last agentic self-report.

Fallback: if no substantial user turn exists, it returns the last 6 turns.

---

## Adapter Tips

| Situation | Recommendation |
|---|---|
| History file is JSONL | Read line-by-line with `bufio.Reader.ReadLine`; skip/truncate lines >5 MB |
| History file is Markdown | Parse role headers line-by-line like `aider_adapter.go` |
| History stored globally (not per-project) | Correlate the session to `workingDir` via metadata, project hash, or cwd before `Detect` returns true |
| Individual content too long | Truncate at 1500 chars: `content[:1500] + "\n... [truncated] ..."` |
| Code blocks in history | Skip lines after 50 inside a code block (see `aider_adapter.go`) |
| Turn selection | Call `selectContext(turns)` — never slice manually |

---

## Good First Issues

Check the [Issues tab](https://github.com/spondanai/aigamesave/issues):

- [ ] **Cline** (VS Code extension — stores history in `.cline/`)
- [ ] **Cursor** — chat history location TBD

---

## Code Style

- No comments explaining *what* the code does — only *why* when non-obvious
- `gofmt` before committing (`go fmt ./...`)
- Keep adapters self-contained — don't add dependencies to `go.mod` unless absolutely necessary
