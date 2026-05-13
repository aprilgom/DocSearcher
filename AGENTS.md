# AGENTS.md

## Project Context
- DocSearcher is a Go 1.24.3 document searcher for indexing HWP/HWPX/PDF files and searching them through a web UI.
- The root module is `hwp-searcher`; the local `goHwpTxt` module is replaced from `./goHwpTxt`.

## Commands
- Setup dependencies: `go mod download`
- Run server: `go run ./cmd/app`
- Run WebView client: `go run ./cmd/client`
- Test all packages: `go test ./...`
- macOS/Linux verification without Windows client: `go test $(go list ./... | grep -v '/cmd/client$')`
- Test local HWP parser module: `cd goHwpTxt && go test ./...`
- Build server: `go build ./cmd/app`

## Navigation
- `cmd/app` - search server entrypoint.
- `cmd/client` - Windows WebView client entrypoint.
- `internal/indexer` - file walking and indexing flow.
- `internal/parser` - HWP/PDF text extraction.
- `internal/search` - Bleve search engine.
- `goHwpTxt` - local HWP/HWPX parser module.

## Git Conventions
- Use Conventional Commits for commit messages: `<type>(<scope>): <subject>`.
- Use the same `type` and `scope` in branch names: `<type>/<scope>-<short-subject>`.
- Write commit message subjects in Korean.
- Keep `<subject>` imperative and concise.
- Keep branch `<short-subject>` lowercase ASCII with hyphens.
- Prefer these types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`, `build`.
- Scope should name the affected module or concern, such as `parser`, `indexer`, `search`, `server`, `client`, `docs`, or `codex`.
- Before creating or renaming a branch, verify the name matches `<type>/<scope>-<short-subject>`; if a requested name does not match, convert it to the closest conforming name.
- Examples:
  - Commit: `docs(codex): git 컨벤션 추가`
  - Branch: `docs/codex-add-git-conventions`
  - Commit: `fix(parser): 빈 pdf 텍스트 처리`
  - Branch: `fix/parser-handle-empty-pdf-text`

## Change Boundaries
- Do not commit local runtime data: `config.json`, `hwp-index.bleve/`, or real test documents under `goHwpTxt/testdata/`.
- Treat `goHwpTxt/pkg/hwp3/hnc2unicode_tables.go` as table data; avoid broad formatting-only edits there.
- Note: `config.example.json` is the committed configuration contract; keep local machine paths only in ignored `config.json`.
- Warning: never commit secrets, private keys, certificate bundles, or `.env*` files. Add placeholder examples instead.
- Warning: destructive commands must not target user document folders. Index reset/recovery code should only remove known runtime index paths such as `hwp-index.bleve/`.
- Important: preserve unrelated dirty worktree changes. Do not use `git reset --hard` or `git checkout --` unless explicitly requested.

## Working Rules
- When asked to "PR 올려" or "올려", create the pull request after pushing the branch; do not stop at reporting the PR creation URL.

## Done Criteria
- For Go code changes, run `go test ./...` unless the current platform cannot build `cmd/client`.
- On macOS/Linux, use `go test $(go list ./... | grep -v '/cmd/client$')` and report that `cmd/client` is Windows-only.
- For parser changes that touch `goHwpTxt`, also run `cd goHwpTxt && go test ./...`.
- For documentation-only changes, Go tests may be skipped; report that no Go code changed.
- If any check cannot be run, report the exact command, reason, and residual risk.
