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
smb_unc       = \\docserver\documents
smb_url       = smb://docserver/documents
```

Rules:

- `root_id` identifies a configured server document root.
- `relative_path` is computed from the root's `server_path`.
- `relative_path` is slash-normalized for storage and API contracts.
- `document_id` is `root_id + ":" + relative_path`.
- `server_path` is server-only operational state for parsing, watching, and
  re-indexing.
- SMB is the primary open backend. Each server document root should describe the
  Samba share that exposes the same root to desktop clients.
- Search results expose `root_id` and `relative_path`, not openable Linux paths.

Validation rules:

- `root_id` must be stable ASCII using lowercase letters, numbers, `_`, and
  `-`. It must not contain `:`, `/`, `\`, whitespace, or empty segments.
- `relative_path` must be non-empty, slash-normalized, and relative to the
  selected configured root.
- `relative_path` must not be absolute, contain Windows drive prefixes, contain
  `..` segments, or contain empty path segments after normalization.
- `document_id` is built only after `root_id` and `relative_path` validation.
  Code that needs to split a `document_id` must split on the first `:` and then
  validate both parts.
- Do not accept client-supplied `server_path` values. Server paths are derived
  from trusted server config plus validated relative paths.

## Server Config

Server-side runtime config describes indexed document roots:

```json
{
  "document_roots": [
    {
      "id": "documents",
      "name": "문서 공유",
      "server_path": "/data/documents",
      "smb_host": "docserver",
      "smb_share": "documents"
    }
  ]
}
```

`document_roots` is shared operational state. It defines what the server scans
and indexes.

Server config rules:

- `id` must satisfy the `root_id` validation rules and be unique.
- `server_path` must be absolute after `filepath.Clean`.
- `smb_host` is the hostname, DNS name, Tailscale name, or IP address clients use
  to reach the Samba server.
- `smb_share` is the Samba share name, which may differ from the final
  `server_path` folder name.
- Windows UNC paths are derived from `smb_host`, `smb_share`, and
  `relative_path`.
- macOS SMB URLs are derived from `smb_host`, `smb_share`, and `relative_path`;
  the desktop client still opens files through the mounted filesystem path.
- Overlapping roots are allowed. If multiple roots contain the same file, the
  most specific matching root owns that file.
- A parent root scan must skip subtrees owned by more specific child roots so
  the same physical file is not indexed under multiple logical IDs.
- `document_roots` is the only supported root configuration for this design.
  Existing `watched_paths` config is not migrated in place.

## Client Config

SMB share metadata in `document_roots` is the primary source for open
resolution. Client-side mount config is an override or local state, not the main
contract.

Windows clients should open search hits through the derived UNC path by default:

```text
\\docserver\documents\shared\2026\sample.hwp
```

Client-side runtime config may map server root IDs to local SMB mount locations
when a machine needs a local override:

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

Client config rules:

- A mount key must match a server `root_id`.
- Windows clients should prefer the server-derived UNC path unless the user
  explicitly chooses a drive-letter or local override.
- Windows mounts may be drive-letter roots such as `Z:\` or UNC roots such as
  `\\docserver\documents`.
- macOS clients use `smb://host/share` to identify or mount the share, then open
  files through local mounted paths such as `/Volumes/documents`.
- The client must join mount roots and `relative_path` with OS path APIs and
  reject any joined result that escapes the configured mount root after
  cleaning.

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
- The desktop client may read root metadata such as `smb_host` and `smb_share`
  from a config/root endpoint, but search hits should remain logical and compact.
- Browser-only usage may show result metadata and copyable paths, but native
  open/reveal actions are available only through a desktop client bridge.

## Open Flow

```text
1. User searches from desktop client.
2. Server returns root_id=documents and relative_path=shared/2026/sample.hwp.
3. Client loads root metadata for documents.
4. Windows clients derive a UNC path from `smb_host`, `smb_share`, and
   `relative_path`; macOS clients find or create a local mount from the SMB URL.
5. Client asks the OS shell to open or reveal the resulting path.
```

Windows:

```text
documents -> \\docserver\documents
shared/2026/sample.hwp -> \\docserver\documents\shared\2026\sample.hwp
```

macOS:

```text
smb://docserver/documents -> /Volumes/documents
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

The client should classify open errors before falling back to a generic failure:

1. Missing root metadata for `root_id`.
2. Missing SMB share metadata or local mount mapping for the current platform.
3. Resolved path escapes the UNC/share/mount root after cleaning.
4. Resolved SMB path does not exist or the share is unavailable.
5. OS permission denial.
6. Shell/default-app open failure.

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

Root ownership rules:

- A file is owned by the most specific configured root that contains it.
- Initial scans, watcher create/write/delete events, and stale-index cleanup
  must all use the same ownership rule.
- If a parent root is scanned, directories owned by more specific child roots
  are skipped during that parent scan. The child root scan is responsible for
  indexing those files.
- Adding a parent root around an existing child root is allowed, but the parent
  root indexes only the files not owned by the child root.
- Removing a child root can move ownership of its files to a parent root; handle
  that as an explicit root-configuration change that requires re-indexing the
  affected roots.

Implementation requirements:

- Keep temporary and lock files excluded by the supported document path policy.
- Keep watcher-based indexing for normal changes.
- Expect Samba clients to save files through temp-file, rewrite, and rename
  patterns.
- Add a periodic rescan if watcher-only indexing proves unreliable under Samba
  saves.
- Watcher events must carry or recover root context so create/write/delete
  events compute the same logical `document_id` as the initial scan.
- Delete events should delete by logical `document_id`; they must not require
  reading a file that no longer exists.
- Store `root_id` and `relative_path` in Bleve. Store `server_path` only if it is
  needed for server-side diagnostics or re-index operations; it must not be
  returned as an openable client path.
