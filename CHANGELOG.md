# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- **Cline adapter** — reads `api_conversation_history.json` from VS Code's `globalStorage/saoudrizwan.claude-dev/tasks/`. The most recent task whose embedded `Current Working Directory` matches the project (or any parent) is selected automatically. Cross-platform: macOS, Linux (`$XDG_CONFIG_HOME`), and Windows (`%APPDATA%`).

---

## [0.3.0] — 2026-04-29

### Added
- `ags save --no-diff` flag — skip embedding the git diff to save tokens on large repos.
- **Cross-platform GitHub Copilot support**: `workspaceStorage` path now resolves correctly on macOS, Linux (`$XDG_CONFIG_HOME`), and Windows (`%APPDATA%`).
- `ActiveFiles` field in `.aigamesave.yaml` — file paths mentioned in conversation (verified to exist on disk) are now surfaced in the resume prompt under `## Active files`.

### Changed
- `SaveGameWithAdapter` refactored into `SaveGameWithOptions(workingDir, SaveOptions{})` for extensibility; old signature kept for backward compatibility.

---

## [0.2.0] — 2026-04-20

### Added
- **GitHub Copilot adapter** — reads VS Code `chatSessions/*.json` from `workspaceStorage`.
- **Smart Anchor Search** (`selectContext`) — shared helper used by all adapters. Backward-searches conversation for the last substantial user instruction (≥10 runes), keeps it as an anchor, appends the most recent 5 turns after it.
- **File path extraction** (`ExtractFilePaths`) — regex-based extraction of mentioned file paths from conversation turns; skips URLs and domain-like strings.
- **Existence check** (`filterExistingPaths`) — only includes paths that resolve to real files under `workingDir`, preventing AI hallucinations.
- **Git file ranking** (`RankFiles`) — scores files by mtime and mention count, returns top 10.
- `ags save --adapter <name>` flag — force a specific adapter; short/partial names resolve via `normalizeAdapterName`.
- `ags load --stdout` flag — print resume prompt to stdout instead of clipboard.
- `ags self-update` command — installs the latest version via `go install`.
- Auto-update check on `save`/`load` (skip with `AGS_SKIP_UPDATE_CHECK=1`).
- Secret redaction (`pkg/redaction`) — masks common API key patterns before saving.

### Changed
- `DetectActiveAdapter` now picks the adapter with the most recently active session when multiple CLIs coexist in the same project.

---

## [0.1.0] — 2026-04-01

### Added
- Initial release with adapters for **Aider**, **Claude Code**, **Gemini CLI**, and **Codex**.
- `ags save` / `ags load` core flow.
- `.aigamesave.yaml` save format with `recentturns` and `gitvision`.
- Auto-add `.aigamesave.yaml` to `.gitignore` on save.
- CI workflow (GitHub Actions).

[Unreleased]: https://github.com/spondanai/aigamesave/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/spondanai/aigamesave/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/spondanai/aigamesave/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/spondanai/aigamesave/releases/tag/v0.1.0
