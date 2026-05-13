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

## Read Order For Codex

1. [Contracts](server-index-smb/contracts.md) - data model, config contract, API
   contract, open flow, indexing rules.
2. [Implementation Plan](server-index-smb/implementation-plan.md) - affected
   code areas, checklist, migration, open questions.
3. [Verification](server-index-smb/verification.md) - server, client, and
   repository verification commands.
4. [Operations And Security](server-index-smb/operations-security.md) - target
   deployment and SMB/Tailscale security model.

## Current Decision

Use a stable logical document identity instead of client-specific or
server-specific absolute paths:

```text
root_id       = documents
relative_path = shared/2026/sample.hwp
document_id   = documents:shared/2026/sample.hwp
server_path   = /data/documents/shared/2026/sample.hwp
```

The API should expose `root_id` and `relative_path`. The server may use
`server_path` internally, but clients must resolve openable file paths from their
own local SMB mount configuration.
