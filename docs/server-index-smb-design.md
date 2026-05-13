# Server Index And SMB Client Open Design

## Codex Brief

DocSearcher should run one server-side index over documents stored on a Linux
server, while desktop clients open search hits through their own SMB mounts.

The important implementation shift is:

```text
Current: document identity == server absolute path
Target:  document identity == root_id + relative_path
```

The server must keep using Linux paths for parsing, watching, and indexing.
Clients must resolve logical search-hit paths to local SMB paths before asking
the operating system to open or reveal a file.

## Goals

- Search Linux-hosted HWP/HWPX/PDF documents from the existing web UI and desktop
  client.
- Keep a single Bleve index on the Linux document server.
- Let Windows and future macOS clients open matched source files in native
  desktop applications through SMB.
- Preserve normal OS file-sharing behavior for editing, saving, locking, and
  permissions.
- Avoid any FTP-style download, upload, edit, and sync lifecycle in DocSearcher.

## Non-Goals

- Do not copy remote files to temporary local files for editing.
- Do not upload edited files back to the server.
- Do not implement custom file locking or conflict resolution.
- Do not bypass Samba, Tailscale, or filesystem permissions.
- Do not make browser-only usage open arbitrary local files directly. Native file
  open is a desktop-client capability.

## Target Deployment

```text
Linux document server
  /data/documents
    Source HWP/HWPX/PDF files
  hwp-index.bleve/
    Single server-side Bleve index
  Samba share
    /data/documents -> \\docserver\documents

Windows client
  Mounts the same share as \\docserver\documents or Z:\
  Opens search hits through the OS shell/default app

macOS client
  Mounts the same share as smb://docserver/documents
  Opens search hits through /Volumes/documents or another mounted path

Tailscale
  Provides private network reachability to the Linux server and SMB port
```

Clients do not build local indexes. The Linux server owns document storage,
filesystem watching, parsing, and indexing.

## Data Model

Use a stable logical document identity instead of client-specific or
server-specific absolute paths.

Current local-machine model:

```text
document_id = /data/documents/shared/2026/sample.hwp
path        = /data/documents/shared/2026/sample.hwp
```

Target central-index model:

```text
root_id       = documents
relative_path = shared/2026/sample.hwp
document_id   = documents:shared/2026/sample.hwp
server_path   = /data/documents/shared/2026/sample.hwp
```

Rules:

- `root_id` identifies a configured server document root.
- `relative_path` is computed from the root's `server_path`.
- `relative_path` is slash-normalized for storage and API contracts.
- `document_id` is `root_id + ":" + relative_path`.
- `server_path` is server-only operational state for parsing, watching, and
  re-indexing.
- Search results expose `root_id` and `relative_path`, not openable Linux paths.

## Configuration Contracts

### Server Config

Server-side runtime config describes indexed document roots:

```json
{
  "document_roots": [
    {
      "id": "documents",
      "name": "문서 공유",
      "server_path": "/data/documents"
    }
  ]
}
```

`document_roots` is shared operational state. It defines what the server scans
and indexes.

### Client Config

Client-side runtime config maps server root IDs to local SMB mount locations:

```json
{
  "mounts": {
    "documents": "Z:\\"
  }
}
```

macOS example:

```json
{
  "mounts": {
    "documents": "/Volumes/documents"
  }
}
```

`mounts` is local machine state because drive letters and mount paths can differ
per user and operating system.

## API Contract

Search results should return logical file identity:

```json
{
  "id": "documents:shared/2026/sample.hwp",
  "root_id": "documents",
  "relative_path": "shared/2026/sample.hwp",
  "fragment": "..."
}
```

Expected behavior:

- The browser UI displays the root name and relative path.
- The desktop client receives `root_id` and `relative_path` for open/reveal
  actions.
- The API must not treat Linux `server_path` values as client-openable paths.

## Open Flow

```text
1. User searches from desktop client.
2. Server returns root_id=documents and relative_path=shared/2026/sample.hwp.
3. Client looks up the local mount for documents.
4. Client joins the mount path and relative path using OS path rules.
5. Client asks the OS shell to open or reveal the resulting path.
```

Windows:

```text
documents -> Z:\
shared/2026/sample.hwp -> Z:\shared\2026\sample.hwp
```

macOS:

```text
documents -> /Volumes/documents
shared/2026/sample.hwp -> /Volumes/documents/shared/2026/sample.hwp
```

Error handling:

- Missing root mount: tell the user the client has no mount configured for that
  root.
- Missing file: tell the user the resolved SMB path does not exist or is not
  currently available.
- Permission failure: report that the OS denied access to the resolved file.
- Open failure: include enough resolved-path context for support without exposing
  unrelated local machine data.

## Indexing And Watch Strategy

The indexer scans each configured server root directly:

```text
root_id=documents
server_path=/data/documents
```

For every supported file under `server_path`, compute:

```text
relative_path = filepath.Rel(server_path, file_path)
document_id   = root_id + ":" + slash-normalized relative_path
```

Implementation requirements:

- Keep temporary and lock files excluded by the supported document path policy.
- Keep watcher-based indexing for normal changes.
- Expect Samba clients to save files through temp-file, rewrite, and rename
  patterns.
- Add a periodic rescan if watcher-only indexing proves unreliable under Samba
  saves.

## Security Model

SMB must not be exposed directly to the public internet. Tailscale should provide
private network reachability, and SMB access should be limited at both the
Tailscale and Samba layers.

Recommended controls:

- Allow SMB only through the private network or Tailscale interface.
- Use Tailscale ACLs or grants to limit access to the document server's SMB port.
- Disable SMBv1.
- Use SMB2/SMB3, with signing and encryption where practical.
- Disable guest and anonymous access.
- Use user or group based Samba permissions.
- Separate read-only users from editors.
- Keep server and clients updated.
- Keep backups or snapshots for document recovery.

DocSearcher must respect filesystem permissions. If SMB access fails, the client
should surface a mount, permission, or availability error.

## Expected Code Impact

This is a medium-size change because the current design assumes `DocumentID` is
the source path.

Likely affected areas:

- `internal/domain`: document identity, search hit fields, document root value
  types.
- `internal/infra/config`: persisted server document roots.
- `internal/usecase`: indexing path-to-document conversion and watch path
  handling.
- `internal/infra/search`: Bleve stored fields and search result hydration.
- `internal/server`: search result fragments and open payloads.
- `web/templates`: result click/double-click behavior.
- `cmd/client`: client-local mount mapping, path resolution, open/reveal
  behavior.
- Tests for domain, config, indexing, search, server fragments, and client path
  joining.

Estimated implementation size:

```text
Central index model and server changes: 500-900 LOC
Windows client mount mapping/open UX:   150-300 LOC
Tests and fixtures:                     300-600 LOC
Total first implementation:             800-1,500 LOC
```

macOS desktop client support is separate because the current `cmd/client` is a
Windows WebView2 entrypoint.

## Implementation Checklist

1. Add domain types for document roots, relative paths, and logical document IDs.
2. Add server-side `document_roots` config.
3. Keep current watched-path behavior only where needed for compatibility.
4. Change indexing to compute relative paths from configured roots.
5. Store `root_id`, `relative_path`, and optional `server_path` in Bleve.
6. Return search hits with logical path fields.
7. Update web result actions to pass `root_id` and `relative_path`.
8. Update the Windows client to resolve root mounts into local SMB paths.
9. Open files through the OS shell/default app.
10. Add "show in folder" behavior.
11. Add clear open failures for missing mounts, unavailable files, and denied
    permissions.
12. Add periodic rescan only if watcher testing under Samba shows missed events.

## Migration Approach

Do not migrate existing Bleve documents in place. The current index stores
absolute paths as IDs, so the simpler and safer migration is:

1. Add document root config.
2. Change indexing to write logical IDs.
3. Reset `hwp-index.bleve`.
4. Re-index configured document roots.

During a transition, existing `watched_paths` config can be treated as legacy
input. New deployments should use `document_roots`.

## Verification Plan

Server-side verification:

- Unit test logical ID creation from `root_id`, `server_path`, and file path.
- Unit test slash-normalized `relative_path` storage.
- Unit test search result hydration includes `root_id` and `relative_path`.
- Integration test re-index after file change under a configured root.

Client-side verification:

- Unit test Windows mount joining from `root_id` and `relative_path`.
- Unit test missing mount, missing file, and permission/open error messages.
- Manual Windows test: search, double-click, edit in native app, save, close, and
  confirm server-side re-index sees the update.

Repository verification:

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

## Open Questions

- Should `document_roots` fully replace `watched_paths`, or should both be
  accepted for one release?
- Should the server store `server_path` in Bleve, or reconstruct it from
  `root_id` and `relative_path` when needed?
- Should client mounts accept UNC paths like `\\docserver\documents` in addition
  to drive-letter paths?
- How should the UI expose mount setup when a desktop client has no mapping for a
  returned `root_id`?
