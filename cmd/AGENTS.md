# AGENTS.md

## Scope
- This directory contains executable entrypoints only.
- Keep application wiring in `cmd/*/main.go`; move reusable behavior to `internal/*`.

## Key Files
- `cmd/app/main.go` - server entrypoint; initializes Bleve at `hwp-index.bleve`, starts the watcher, then starts the web server on port `8080`.
- `cmd/client/main.go` - Windows WebView2 client entrypoint; reads `server.txt`, opens the UI, and delegates file opening through `cmd /c start`.
- `cmd/client/server.txt` - packaged/default server URL for the WebView client.

## Commands
- Run server: `go run ./cmd/app`
- Run WebView client: `go run ./cmd/client`
- Test all packages: `go test ./...`

## Platform Notes
- `cmd/client` depends on `github.com/jchv/go-webview2` and Windows shell behavior.
- On macOS, `go test ./...` may fail because the Windows WebView client does not build there. If only non-client code changed, verify the affected package set and report the `cmd/client` limitation explicitly.

## Safety And Change Boundaries
- Do not commit runtime output from the server, especially `hwp-index.bleve/` or local `config.json`.
- Avoid adding business logic, parsing, indexing, or search behavior to `cmd`; keep entrypoints thin.
- Do not change the default port, index path, or client URL behavior without checking the matching `internal/server`, `internal/search`, and packaging assumptions.

## Done Criteria
- For `cmd/app` changes, run `go test ./internal/...` and, when practical, `go run ./cmd/app` to confirm startup.
- For `cmd/client` changes, run or build on Windows when possible; on macOS, document that WebView2 verification was skipped because the client is Windows-specific.
- Report exact commands run and any skipped verification.
