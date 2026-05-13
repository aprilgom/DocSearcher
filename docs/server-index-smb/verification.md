# Server Index SMB Verification

## Server Verification

- Unit test logical ID creation from `root_id`, `server_path`, and file path.
- Unit test slash-normalized `relative_path` storage.
- Unit test search result hydration includes `root_id` and `relative_path`.
- Integration test re-index after file change under a configured root.

## Client Verification

- Unit test Windows mount joining from `root_id` and `relative_path`.
- Unit test missing mount, missing file, and permission/open error messages.
- Manual Windows test: search, double-click, edit in native app, save, close, and
  confirm server-side re-index sees the update.

## Repository Verification

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
