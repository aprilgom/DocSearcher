# AGENTS.md

## Scope
- This directory contains the application implementation behind the `cmd` entrypoints.
- Keep package APIs small and prefer package-local helpers unless behavior must be shared across packages.

## Key Files
- `internal/config/config.go` - configuration loading and persisted settings.
- `internal/indexer/walker.go` - file walking and indexing flow.
- `internal/parser/parser.go` - HWP/HWPX/PDF text extraction dispatch.
- `internal/search/engine.go` - Bleve index setup, indexing, querying, and close behavior.
- `internal/server/server.go` - HTTP server and web UI handlers.
- `internal/watcher/watcher.go` - filesystem watching and re-indexing trigger.

## Commands
```bash
go mod download
go test ./internal/...
go test ./internal/parser
go run ./cmd/app
```
- Full repo test when platform allows it: `go test ./...`

## Platform Notes
- `go test ./internal/...` is the preferred macOS-safe verification for internal-only changes.
- Warning: full `go test ./...` may fail on macOS because `cmd/client` uses the Windows WebView2 client. Report that limitation if it prevents full verification.

## Known Failures
- `go test ./...` can fail on macOS outside `internal` because `cmd/client` is Windows-specific; this is not an internal package failure.

## Dependencies
- See [../ARCHITECTURE.md](../ARCHITECTURE.md) for cross-module data flow.
- `internal/server` should depend on injected interfaces and `internal/domain`, not concrete config/search/indexer/watcher adapters.
- `internal/indexer` should depend on injected indexing behavior and `internal/domain`, not concrete parser/search adapters.
- Parser behavior changes can affect indexed content and search results through the `cmd/app` wiring path.

## Safety And Change Boundaries
- Do not commit runtime data such as `config.json`, `hwp-index.bleve/`, or real documents under `goHwpTxt/testdata/`.
- Treat `goHwpTxt` as an external local replacement module; avoid editing it for internal package work unless explicitly asked.
- Be careful with destructive filesystem operations in index recovery, watching, and config code. Restrict deletes to known runtime paths and preserve user documents.
- Keep parser tests synthetic or fixture-based; do not add private or real HWP/PDF documents.

## Done Criteria
- Run the narrow package test for the changed package, then `go test ./internal/...`.
- Run `go test ./...` when changes affect entrypoint wiring or cross-package behavior; if macOS blocks it through `cmd/client`, report the exact failure reason.
- Report exact commands run and any skipped verification.
