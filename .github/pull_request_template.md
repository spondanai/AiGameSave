## Summary
<!-- One-line description of what this PR does -->

## Type
- [ ] New adapter (`feat: add <ToolName> adapter`)
- [ ] Bug fix
- [ ] Refactor / performance
- [ ] Documentation

## For new adapters — checklist
- [ ] History file path and format documented in PR description
- [ ] Sample raw history snippet included (redact secrets)
- [ ] `Detect()` does not return `true` for unrelated projects
- [ ] `Extract()` calls `selectContext(turns)` — no manual slicing
- [ ] Content truncated at 1 500 chars per turn
- [ ] `LastActive()` implemented (for multi-CLI tie-breaking)
- [ ] Adapter registered in `registry.go`
- [ ] `go test ./...` passes
- [ ] README table updated

## Testing
<!-- How did you verify this works? Output of `ags save` on a real project? -->
