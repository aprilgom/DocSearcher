# Repository Verification

Documentation-only edits:

- Review rendered Markdown and `git diff`.
- Go tests may be skipped when no Go code or config behavior changed. Report
  that no Go code changed.

On macOS/Linux:

```bash
go test $(go list ./... | grep -v '/cmd/client$')
go build ./cmd/app
```

On Windows, also run:

```bash
go run ./cmd/client
```

If `goHwpTxt` is touched, also run:

```bash
cd goHwpTxt && go test ./...
```
