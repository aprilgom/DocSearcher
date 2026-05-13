# Server Index SMB Contracts

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

## Server Config

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

## Client Config

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

## Indexing And Watch Rules

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
