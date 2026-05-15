# Server Index SMB Design

This folder defines the design for moving DocSearcher from local absolute-path
document IDs to a central Linux index with SMB-openable desktop paths.

The goal is one server-side index over Linux files, while Windows and macOS
desktop clients open the same documents through Samba/SMB paths.

## Read Order

1. [Contracts](contracts.md) - data model, config, API, open flow, indexing, and
   Bleve storage contracts.
2. [Implementation Plan](implementation-plan.md) - affected code areas,
   decisions, phased tasks, migration, and remaining questions.
3. [Verification](verification.md) - test cases and platform-specific manual
   checks.
4. [Operations And Security](operations-security.md) - deployment shape,
   Samba/Tailscale assumptions, and security controls.

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
- Keep `server_path` out of search responses, root metadata responses, browser
  copy actions, and desktop open payloads. If `server_path` is stored in Bleve,
  it is stored-only diagnostics and must never be indexed, highlighted,
  queryable, or returned to clients.
- Validate `relative_path` before indexing. If a file's relative path is not
  valid UTF-8 text or is not SMB-open-safe, including any control character,
  `\`, Windows-illegal filename character (`*`, `?`, `"`, `<`, `>`, `|`), or
  reserved device basename (`CON`, `PRN`, `AUX`, `NUL`, `COM1`-`COM9`,
  `LPT1`-`LPT9`, including with extensions), or any path segment ending in a
  space or `.`, skip that file and log an operational warning instead of
  encoding, replacing, or rewriting the path.
- Validate `smb_host` and `smb_share` as UNC/SMB URL components: trim them,
  reject empty values, reject slashes, backslashes, whitespace, and control
  characters, reject `%`, non-ASCII text, and `:` in `smb_host`, reject
  UNC/path metacharacters in `smb_share` (`:`, `*`, `?`, `"`, `<`, `>`, `|`),
  and keep `smb_share` to a single share name. Only ASCII hostnames, IP
  addresses, and Tailscale names are supported in Milestone 1; IDNA, ports, and
  IPv6 literals are unsupported until explicitly normalized later.
- `smb_aliases` is optional server-root metadata shaped as an array of
  `{ "host": "...", "share": "..." }` pairs. Aliases are operator-owned server
  config because they authorize alternate shares or DFS names for the same root.
  Alias host/share values follow the same trimming, validation, and
  normalization rules as `smb_host` and `smb_share`.
- Reject duplicate canonical `server_path` document roots during config
  validation. Parent/child overlaps remain valid when the canonical roots differ.
- Treat Samba share mapping as operator-owned config. DocSearcher should not
  parse Samba config files.
- Use Wails for new cross-platform desktop behavior. Keep `cmd/client` as the
  current Windows WebView2 client until it is intentionally retired.
- If root metadata is missing, stale, or inconsistent with a search hit, refresh
  root metadata once for known stale-cache failure classes, then fail open/reveal
  with a clear client error instead of guessing a path. `GET
  /api/document-roots` returns a root metadata `revision`; search
  responses/fragments carry the revision used for rendering, and clients refresh
  root metadata before open/reveal when their cached revision is missing or
  differs. The revision is server-generated and opaque for the whole root
  metadata set; it changes on any root metadata or validation-canonicalization
  change, and clients compare it only for equality.
- Desktop bridge and browser copy code must revalidate raw `relative_path`
  values received from API, HTMX, DOM, or logical-ID payloads before deriving
  UNC paths, SMB URLs, or mounted paths. Treat the server as the source of
  truth, but fail closed on hostile or stale client payloads instead of trusting
  cached DOM/API data. Client revalidation must also reject non-UTF-8 text,
  control characters, `.` segments such as `a/./b.hwp`, and empty segments such
  as `a//b.hwp` instead of slash-normalizing malformed payloads.
- macOS mount discovery and client-local mount overrides must verify that the
  selected mount root matches the expected SMB host/share when the OS exposes
  mount identity. If the match cannot be verified or differs, fail with a clear
  mount-unverified or mount-mismatch error instead of opening from that path.
- Windows UNC overrides must match the expected normalized SMB host/share or any
  normalized `smb_aliases` pair for that root. Drive-letter overrides remain
  explicit local config and containment-only.
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
