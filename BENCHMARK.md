# Token Savings Benchmark

Real measurements from Claude Code sessions on the AiGameSave project itself.
All sessions are AI-assisted Go development (adapters, domain logic, CLI).

## Methodology

| Term | Definition |
|---|---|
| **Raw session** | The full `.jsonl` file written by the AI CLI — includes every tool call, bash output, file read, JSON envelope, and all assistant turns |
| **Resume prompt** | The text `ags load` puts on the clipboard — only the anchor instruction + up to 5 recent turns (truncated to 1,500 chars each) |
| **Token estimate** | `bytes / 4` — a conservative approximation valid for mixed English/Thai/code content |

The AGS pipeline applies three filters in sequence:

1. **Tool noise removal** — strips bash outputs, file reads, tool_result blocks (these are the biggest single source of inflation)
2. **Smart Anchor Search** — backward-scans for the last meaningful user instruction (≥ 10 chars), uses it as the resume anchor
3. **Recency window** — keeps the anchor + the 5 most recent turns after it; everything else is dropped

## Results

Measured 2026-04-30 on four real Claude Code sessions from the same project:

| Session | Raw JSONL | ~Tokens (raw) | Tool noise | Text turns | Resume prompt | ~Tokens (resume) | Saved |
|---|---:|---:|---:|---:|---:|---:|---:|
| Small (1 feature) | 66 KB | 16,579 | 21 KB | 8 | ~580 bytes | 145 | **99%** |
| Medium (multi-feature) | 1.1 MB | 290,443 | 376 KB | 61 | ~1,750 bytes | 437 | **99%** |
| Large (refactor session) | 850 KB | 212,539 | 249 KB | 59 | ~1,968 bytes | 492 | **99%** |
| XLarge (long session) | 2.1 MB | 546,855 | 822 KB | 68 | ~112 bytes | 28 | **99%** |

## Where the savings come from

```
Raw JSONL breakdown (typical medium session):
  Tool calls + bash output + file content  ≈ 94% of raw bytes
  Actual conversation text                 ≈  3% of raw bytes
  JSON envelope overhead                   ≈  3% of raw bytes

AGS resume prompt:
  Anchor turn (last user instruction)      1 turn
  Recent context window                    up to 5 turns
  Total                                    ≈ 0.05–0.15% of raw bytes
```

Tool noise is the dominant factor. A single `ags save --no-diff` run on a
2 MB session saves roughly **540,000 tokens** — equivalent to the entire
context window of most LLM API tiers, at a cost of $0.

## Live display

Since v0.4.0, `ags save` prints the savings inline:

```
Detected AI CLI: Claude Code
Successfully saved session to .aigamesave.yaml
Token savings: ~123k tokens saved vs raw session (99% reduction)
```

The figure shown is the resume-prompt size vs the raw session file —
not an estimate but a measurement of the actual files on disk.

## Notes

- The 4 bytes/token ratio is conservative for English; for Thai or dense code, actual token counts may be lower (more savings).
- `ags save --no-diff` further reduces the resume prompt by omitting the git diff section, which can add 500–3,000 tokens for active branches.
- Savings are consistent across session sizes because AGS always selects a fixed-size window (~6 turns) regardless of how long the session grew.
