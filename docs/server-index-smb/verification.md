# Server Index SMB Verification

## Server Verification

- Unit test logical ID creation from `root_id`, `server_path`, and file path.
- Unit test `root_id` rejects `:`, slashes, whitespace, empty values, and
  uppercase or non-ASCII values.
- Unit test `relative_path` rejects absolute paths, Windows drive prefixes,
  `..` segments, and empty path segments.
- Unit test slash-normalized `relative_path` storage.
- Unit test logical ID parsing splits on the first `:` and revalidates both
  fields.
- Unit test document root config validates required `smb_host` and `smb_share`
  metadata for SMB-open-enabled roots.
- Unit test search result hydration includes `root_id` and `relative_path`.
- Unit test overlapping roots choose the most specific matching root.
- Unit test parent-root scans skip subtrees owned by more specific child roots.
- Integration test overlapping roots do not produce duplicate logical documents
  during a full scan.
- Integration test removing a child root and re-indexing moves affected files to
  the parent root when a parent root still contains them.
- Unit test legacy `watched_paths` load as transition document roots when
  `document_roots` is absent.
- Unit test legacy `watched_paths` roots are marked as scan-compatible but not
  SMB-open-ready until explicit Samba share metadata is configured.
- Integration test re-index after file change under a configured root.
- Integration test delete events remove the logical document ID without needing
  the deleted file to exist.

## Client Verification

- Unit test Windows mount joining from `root_id` and `relative_path`.
- Unit test Windows UNC path derivation from `smb_host`, `smb_share`, and
  `relative_path`.
- Unit test Windows UNC mount joining from `root_id` and `relative_path`.
- Unit test macOS mount joining from `root_id` and `relative_path`.
- Unit test macOS SMB URL derivation from `smb_host` and `smb_share`.
- Unit test path joining rejects cleaned paths that escape the configured mount
  root.
- Unit test missing mount, missing file, and permission/open error messages.
- Manual Windows test: search, double-click, edit in native app, save, close, and
  confirm server-side re-index sees the update.
- Manual Windows UNC test: configure `\\docserver\documents`, open a search hit,
  reveal it in Explorer, edit, save, and confirm the server re-indexes it.
- Manual macOS test after Wails support exists: configure `/Volumes/documents`,
  open a search hit, reveal it in Finder, edit, save, and confirm the server
  re-indexes it.

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
