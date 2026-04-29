# 💾 AiGameSave (AGS)

**Zero-LLM context preservation tool for AI CLI Developers.**

AiGameSave (AGS) acts like a "save point" for your AI coding sessions. It extracts your current context (recent conversation history + modified files) and saves it as a lightweight YAML file. When you start a new session, you can "load" this save to bring your AI up to speed instantly—**without wasting massive amounts of tokens re-scanning the entire project.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/mrporing/aigamesave)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

## ✨ Why AGS?

AI coding assistants are powerful, but starting a new session often means losing context—or spending dollars on input tokens just to re-read the workspace map. In agentic workflows this gets worse: the AI's own self-talk (running commands, checking logs, summarising results) can push your original instruction clean out of the context window.

AGS solves this by:
- **Zero Token Cost:** Uses 100% local Go logic (Git + file parsing). No API calls.
- **Instruction Anchoring:** Backward-searches for your last meaningful instruction so agentic self-talk can't evict it.
- **Smart Truncation:** Automatically truncates massive code blocks and terminal outputs from the history.
- **Auto-Redaction:** Sanitises common API keys before saving to prevent accidental leaks.
- **Plug-and-Play Architecture:** Incredibly easy to add support for new AI CLIs.

## 🚀 Supported AI CLIs

| Tool | Session location |
|---|---|
| **Aider** | `.aider.chat.history.md` |
| **Claude Code** | `~/.claude/projects/<encoded-dir>/*.jsonl` |
| **Gemini CLI** | `~/.gemini/tmp/<user>/chats/*.jsonl` (correlated by `projectHash`) |
| **Codex** | `~/.codex/sessions/**/*.jsonl` (correlated by session `cwd`) |
| **GitHub Copilot** (VS Code) | `~/Library/Application Support/Code/User/workspaceStorage/<hash>/chatSessions/*.json` |
| *Cline* | ⏳ Coming soon — PRs welcome |
| *Cursor* | ⏳ Coming soon — PRs welcome |

---

## 📦 Installation

Ensure you have Go installed, then run:

```bash
go install github.com/spondanai/aigamesave/cmd/ags@latest
```

AGS checks for newer versions when you run `ags save` or `ags load`. If an update is available:

```bash
ags self-update
```

For offline or CI usage, disable the check with `AGS_SKIP_UPDATE_CHECK=1`.

---

## 🎮 Usage

### 1. Save Your Game

Before ending your session, run:

```bash
ags save
```

AGS detects your active AI CLI, extracts conversation context, runs a Git vision check to find modified files, redacts secrets, and saves everything into a `.aigamesave.yaml` file (automatically added to `.gitignore`).

**Context selection:** AGS does not blindly grab the last N turns. It backward-searches for your last meaningful instruction (≥10 characters), then keeps that anchor plus the most recent work the AI did afterwards. Short acknowledgements like "ok" or "ลองรันดู" are skipped so the anchor stays on your actual goal.

If you use multiple AI CLIs in the same project, AGS picks the one with the most recently active session. Override manually:

```bash
ags save --adapter codex
ags save --adapter gemini
ags save --adapter claude
ags save --adapter copilot
```

Short aliases work too: `--adapter github` or `--adapter githubcopilot` also resolve to GitHub Copilot.

### 2. Load Your Game

When you return, run:

```bash
ags load
```

AGS reads the save file, formats a concise prompt, and **copies it directly to your clipboard**. Paste it into your AI CLI and say: *"Let's continue."*

Use `ags load --stdout` to pipe the output instead of copying to clipboard.

---

## 🛠️ How It Works

1. **Detect Adapter** — Scans for known AI history files; picks the one touched most recently.
2. **Anchor & Extract** — Finds the last substantial user instruction, keeps that + the most recent 5 turns after it.
3. **Git Vision** — Runs `git status --porcelain` to identify files currently being worked on.
4. **Redact & Save** — Strips common secret patterns, then writes `.aigamesave.yaml`.

---

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide. The short version:

1. Create `internal/adapters/<name>_adapter.go` and implement `HistoryExtractor`.
2. Add your adapter to `registry.go`.
3. Call `selectContext(turns)` (from `context.go`) instead of slicing turns manually.
4. Open a PR. 🎉

### Good First Issues
Check the [Issues tab](https://github.com/spondanai/aigamesave/issues). We are actively looking for adapters for **Cline** and **Cursor**.

---

## 📜 License

MIT License. See [LICENSE](LICENSE) for more information.
