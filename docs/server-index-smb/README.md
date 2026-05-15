# Server Index SMB Design

This folder defines the design for moving DocSearcher from local absolute-path
document IDs to a central Linux index with SMB-openable desktop paths.

The goal is one server-side index over Linux files, while Windows and macOS
desktop clients open the same documents through Samba/SMB paths.

## Read Order

1. [Contracts](contracts.md) - data model, config, API, open flow, indexing, and
   Bleve storage contract map. Detailed contract files live under
   [`contracts/`](contracts/).
2. [Implementation Plan](implementation-plan.md) - affected code areas,
   decisions, phased tasks, migration, and remaining questions map. Detailed
   plan files live under [`implementation/`](implementation/).
3. [Verification](verification.md) - test cases and platform-specific manual
   checks map. Detailed verification files live under
   [`verification/`](verification/).
4. [Operations And Security](operations-security.md) - deployment shape,
   Samba/Tailscale assumptions, and security controls map. Detailed operations
   files live under [`operations/`](operations/).

## Split Documents

Contract details:

- [Data Model](contracts/data-model.md)
- [Server Config](contracts/server-config.md)
- [Client Config](contracts/client-config.md)
- [API Contract](contracts/api-contract.md)
- [Open Flow](contracts/open-flow.md)
- [Indexing And Watch Rules](contracts/indexing-watch.md)
- [Bleve Storage Contract](contracts/bleve-storage.md)
- [Cross-Platform Name Caveats](contracts/name-caveats.md)

Implementation details:

- [Overview And Milestones](implementation/overview.md)
- [Implementation Tasks](implementation/tasks.md)
- [Migration And Questions](implementation/migration.md)

Verification details:

- [Server Verification](verification/server.md)
- [Client Verification](verification/client.md)
- [Repository Verification](verification/repository.md)

Operations details:

- [Target Deployment](operations/deployment.md)
- [Deployment Runbook](operations/runbook.md)
- [Operational Diagnostics](operations/diagnostics.md)
- [Security Model](operations/security.md)
- [Data Exposure Rules](operations/data-exposure.md)

## Core Decision

The server indexes Linux filesystem paths, but clients open files through SMB.
Do not expose server-local absolute paths as client-openable paths, API
contracts, browser copy actions, or desktop open payloads.

```text
document_id   = root_id + ":" + relative_path
root_id       = documents
relative_path = shared/2026/sample.hwp
server_path   = /data/documents/shared/2026/sample.hwp
windows open  = \\docserver\documents\shared\2026\sample.hwp
macOS open    = /Volumes/documents/shared/2026/sample.hwp
```

`server_path` is operational state. `root_id`, `relative_path`, and root SMB
metadata are the client contract.

## Target Shape

Server config names document roots explicitly:

```json
{
  "document_roots": [
    {
      "id": "documents",
      "name": "문서 공유",
      "server_path": "/data/documents",
      "smb_host": "docserver",
      "smb_share": "documents",
      "smb_aliases": [
        { "host": "dfs-docs", "share": "documents" }
      ]
    }
  ]
}
```

Search results identify documents with logical fields:

```json
{
  "id": "documents:shared/2026/sample.hwp",
  "root_id": "documents",
  "root_name": "문서 공유",
  "root_revision": "roots-0007",
  "relative_path": "shared/2026/sample.hwp"
}
```

Desktop clients resolve `root_id` through `GET /api/document-roots`, refresh
that metadata when their cached `revision` is missing or differs from
`root_revision`, then join the returned SMB root metadata with `relative_path`.

## Rollout Order

1. Replace server-side `watched_paths` behavior with validated
   `document_roots`.
2. Index documents using logical IDs and store `root_id` plus `relative_path` in
   Bleve.
3. Return logical path fields from search responses and HTMX fragments.
4. Add `GET /api/document-roots` with client-safe SMB metadata.
5. Reset and rebuild `hwp-index.bleve`.
6. Enable desktop open/reveal only when the active client can consume
   `root_id` and `relative_path`.
7. Implement Windows UNC open support, then Wails-based macOS support.

Each rollout slice must preserve browser search without exposing
`server_path`. Native open/reveal is optional until a desktop bridge explicitly
supports the logical payload.

## Non-Goals

- Do not migrate existing absolute-path Bleve IDs in place.
- Do not silently convert legacy `watched_paths` into document roots.
- Do not parse Samba config files from DocSearcher.
- Do not treat Linux `server_path` as a client-openable path.
- Do not make macOS support by extending the Windows-only WebView2 client.

## Implementation Guardrails

- Implement `document_roots` before changing open behavior.
- Reset and rebuild `hwp-index.bleve` when switching from absolute path IDs to
  logical IDs.
- Keep `server_path` server-only; see
  [Bleve Storage Contract](contracts/bleve-storage.md) for diagnostic storage
  limits.
- Validate `relative_path`, SMB metadata, aliases, and root containment before
  indexing or opening; see [Data Model](contracts/data-model.md),
  [Server Config](contracts/server-config.md), and
  [Client Config](contracts/client-config.md).
- Treat Samba share mapping as operator-owned config. DocSearcher does not parse
  Samba config files.
- Use Wails for new cross-platform desktop behavior. Keep `cmd/client` as the
  current Windows WebView2 client until it is intentionally retired.
- Refresh root metadata by `revision`, revalidate raw client payloads, and fail
  closed instead of guessing open paths; see [API Contract](contracts/api-contract.md)
  and [Open Flow](contracts/open-flow.md).
- Verify mount/override identity before native open where possible.
- Do not use the legacy `openFile(path)` bridge as a fallback after the server
  stops returning client-local absolute paths.

## Acceptance Checklist

- Search results display root name plus relative path, not server absolute path.
- `GET /api/document-roots` returns `revision` plus roots containing `id`,
  `name`, `smb_host`, `smb_share`, and optional client-safe `smb_aliases`, and
  never returns `server_path`.
- Browser-only UI can search and copy safe logical or SMB-derived paths without
  exposing server-local paths.
- Browser copy and desktop open/reveal reject hostile or stale client
  `relative_path` payloads before path derivation.
- Native open/reveal actions are hidden or disabled unless a desktop bridge is
  available and supports the logical open payload.
- Windows open resolves to UNC paths such as
  `\\docserver\documents\shared\2026\sample.hwp`.
- macOS open resolves through a mounted SMB share such as
  `/Volumes/documents/shared/2026/sample.hwp`.
- Index rebuild after the migration produces logical Bleve document IDs.
