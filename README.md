# 💾 AiGameSave (AGS)

**Zero-LLM context preservation tool for AI CLI Developers.**

AiGameSave (AGS) acts like a "save point" for your AI coding sessions. It extracts your current context (recent conversation history + modified files) and saves it as a lightweight YAML file. When you start a new session the next day, you can "load" this save to bring your AI up to speed instantly—**without wasting massive amounts of tokens re-scanning the entire project.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/mrporing/aigamesave)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

## ✨ Why AGS?

AI coding assistants (like Aider, Claude Code, or Cline) are powerful, but starting a new session often means losing context or spending dollars on input tokens just to re-read the workspace map.

AGS solves this by:
- **Zero Token Cost:** Uses 100% local Go logic (Git + File parsing). No API calls.
- **Smart Truncation:** Automatically truncates massive code blocks and terminal outputs from the history.
- **Auto-Redaction:** Sanitizes common API keys before saving to prevent accidental leaks.
- **Plug-and-Play Architecture:** Incredibly easy to add support for new AI CLIs.

## 🚀 Supported AI CLIs
- ✅ **Aider** (`.aider.chat.history.md`)
- ✅ **Claude Code** (`.claude/history.jsonl`)
- ✅ **Gemini CLI** (`.gemini_history.jsonl` or `.gemini/history.jsonl`)
- ⏳ *Cline (Coming Soon - PRs welcome!)*
- ⏳ *Cursor (Coming Soon)*

---

## 📦 Installation

Ensure you have Go installed, then run:

```bash
go install github.com/spondanai/AiGameSave/cmd/ags@latest
```

## 🎮 Usage

### 1. Save Your Game
Before closing your terminal for the day, run:
```bash
ags save
```
*What happens:* AGS detects your AI CLI, extracts the last ~3 conversation turns, runs a Git vision check to find modified files, redacts secrets, and saves everything into a tiny `.aigamesave.yaml` file (which is automatically added to `.gitignore`).

### 2. Load Your Game
When you return, simply run:
```bash
ags load
```
*What happens:* AGS reads the save file, formats a concise prompt, and **copies it directly to your clipboard**. 

Just paste it into your AI CLI and say: *"Let's continue."*

*(If you prefer to pipe the output, use `ags load --stdout`)*

---

## 🛠️ How it Works

AGS acts as a "Hard Support" that places wards (vision) for your AI. It follows a Clean Architecture approach:
1. **Detect Adapter:** Scans the directory for known AI history files.
2. **Extract & Clean:** Reads the history, truncates long blocks (>50 lines), and keeps only the most relevant recent turns.
3. **Git Vision:** Runs `git status --porcelain` to identify files currently being worked on.
4. **Package:** Combines this into a highly optimized prompt ready for the next session.

---

## 🤝 Contributing (Adding a New AI Adapter)

AGS is built with a **Registry Pattern**, making it incredibly easy for anyone to add support for a new AI tool. You only need to create **one file**!

### Step-by-Step Guide:

1. **Create an Adapter File:**
   Create a new file in `internal/adapters/` (e.g., `cline_adapter.go`).

2. **Implement the Interface:**
   Your struct just needs to implement the `HistoryExtractor` interface from `internal/domain/extractor.go`:

   ```go
   package adapters
   
   import "github.com/spondanai/AiGameSave/internal/domain"

   type MyAIAdapter struct{}

   func NewMyAIAdapter() *MyAIAdapter { return &MyAIAdapter{} }

   // Detect checks if this AI CLI is used in the current directory
   func (a *MyAIAdapter) Detect(workingDir string) bool {
       // e.g., check for a specific config folder
       return true
   }

   // Extract parses the history and returns the SessionState
   func (a *MyAIAdapter) Extract(workingDir string) (domain.SessionState, error) {
       // 1. Read the specific history file
       // 2. Parse turns (User/Assistant)
       // 3. Truncate long content
       // 4. Return the last ~3 turns
       return domain.SessionState{RecentTurns: turns}, nil
   }

   func (a *MyAIAdapter) Name() string { return "MyAwesomeAI" }
   ```

3. **Register It:**
   Open `internal/adapters/registry.go` and add your adapter to the `init()` function:
   ```go
   func init() {
       registry = append(registry, 
           NewAiderAdapter(), 
           NewClaudeAdapter(),
           NewMyAIAdapter(), // <-- Add yours here!
       )
   }
   ```

4. **Submit a PR!** 🎉

### Good First Issues
If you are looking to contribute, check our [Issues](https://github.com/spondanai/AiGameSave/issues) tab. We are actively looking for help building adapters for **Cline**, **Cursor**, and improving our **Git Ranking Heuristics**.

---

## 📜 License

MIT License. See [LICENSE](LICENSE) for more information.
