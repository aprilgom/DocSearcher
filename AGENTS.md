# AGENTS.md

## Project Context
- DocSearcher is a Go 1.24.3 document searcher for indexing HWP/HWPX/PDF files and searching them through a web UI.
- The root module is `hwp-searcher`; the local `goHwpTxt` module is replaced from `./goHwpTxt`.

## Commands
- Run server: `go run ./cmd/app`
- Run WebView client: `go run ./cmd/client`
- Test all packages: `go test ./...`

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
- Examples:
  - Commit: `docs(codex): git 컨벤션 추가`
  - Branch: `docs/codex-add-git-conventions`
  - Commit: `fix(parser): 빈 pdf 텍스트 처리`
  - Branch: `fix/parser-handle-empty-pdf-text`

## Change Boundaries
- Do not commit local runtime data: `config.json`, `hwp-index.bleve/`, or real test documents under `goHwpTxt/testdata/`.
- Treat `goHwpTxt/pkg/hwp3/hnc2unicode_tables.go` as table data; avoid broad formatting-only edits there.

## Done Criteria
- Run `go test ./...` before reporting completion when Go code changes.
- If a check cannot be run, report the reason and the residual risk.
