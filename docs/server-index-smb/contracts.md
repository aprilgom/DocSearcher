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
  selected configured root. Slash normalization is only for controlled server
  filesystem-to-relative-path computation.
- `relative_path` must be valid UTF-8 text and must not contain control
  characters. Invalid UTF-8 and control characters are rejected; they are not
  encoded, replaced, or normalized into another name.
- Raw API, DOM, or logical-ID payloads for `relative_path` must not be absolute,
  contain Windows drive prefixes, contain `..` or `.` segments, contain `:`,
  contain `\`, contain Windows-illegal filename characters (`*`, `?`, `"`,
  `<`, `>`, `|`), contain empty path segments such as `a//b.hwp`, or contain
  any segment ending in a space or `.`. Reject malformed client payloads before
  deriving paths; do not normalize them into an acceptable value. In
  particular, reject `a/./b.hwp` as malformed rather than cleaning it to
  `a/b.hwp`. This keeps logical IDs unambiguous and avoids
  Windows-incompatible names in the SMB open path.
  Backslashes are rejected because Windows UNC APIs treat them as path
  separators, so accepting them as literal filename text would change the
  opened path. Trailing segment spaces and dots are rejected because Windows
  file APIs and UNC opens may trim or reject those names.
- Each `relative_path` segment must not have a reserved Windows device basename:
  `CON`, `PRN`, `AUX`, `NUL`, `COM1`-`COM9`, or `LPT1`-`LPT9`, case-insensitive
  and including names with extensions such as `CON.txt`.
- `document_id` is built only after `root_id` and `relative_path` validation.
  Code that needs to split a `document_id` must split on the first `:` and then
  validate both parts.
- `document_id` is not URL-encoded. Store it as `root_id`, a literal `:`, and
  the slash-normalized UTF-8 `relative_path`. Apply URL escaping only when
  deriving SMB URLs for copy or mount flows.
- Do not accept client-supplied `server_path` values. Server paths are derived
  from trusted server config plus validated relative paths.
- Clients must not reconstruct `server_path` from logical fields or submit a
  guessed server path back to the server for open, reveal, delete, or re-index
  actions.

Path containment rules:

- Never use string-prefix checks to decide whether a file is contained by a
  root. Containment is path-component based.
- On Unix-like platforms, clean both paths, compute `filepath.Rel(root, file)`,
  and accept only non-absolute results where `rel` is neither `..` nor prefixed
  by `../`. Names such as `..draft/file.hwp` are valid contained paths when the
  component check passes.
- The root itself must resolve successfully during config validation. If
  `server_path` is a symlink, resolve it with `EvalSymlinks` and use that
  canonical root for containment and relative-path computation. A root symlink
  that cannot be resolved is a config error.
- Files that disappear during scan or watch handling may be skipped and retried
  by a later event or reconciliation scan.
- Symlink entries are not followed and are not indexed during scans or watcher
  handling. This avoids duplicate logical documents for the same physical file
  and prevents symlink escapes outside the configured root.
- A symlink entry is skipped, not treated as a fatal root error, because a
  single user-created or tool-created link must not stop indexing the rest of
  the root.
- On Windows, containment must compare path components case-insensitively after
  cleaning. Drive roots or UNC share roots must match before relative path
  comparison.
- Windows drive and UNC prefix traps must be rejected: `Z:\docs2` is not under
  `Z:\docs`, and `\\server\share2` is not under `\\server\share`.

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
      "smb_share": "documents",
      "smb_aliases": [
        { "host": "dfs-docs", "share": "documents" }
      ]
    }
  ]
}
```

`document_roots` is shared operational state. It defines what the server scans
and indexes.

Minimal valid examples:

```text
id:          documents
name:        문서 공유
server_path: /data/documents
smb_host:    docserver
smb_share:   documents
```

Invalid examples:

```text
id:          Documents        # uppercase
id:          docs/main        # slash
relative:    ../secret.hwp    # parent escape
relative:    C:/docs/a.hwp    # Windows drive prefix
relative:    a:b.hwp          # ambiguous document_id separator
relative:    a\b.hwp          # backslash changes UNC path semantics
relative:    folder./a.hwp     # Windows file APIs/UNC may trim or reject segment
smb_host:    user@docserver   # credentials/authority syntax
smb_host:    docserver:445     # ports are not supported in Milestone 1
smb_host:    [fd7a::1]         # IPv6 literals are not supported in Milestone 1
smb_host:    docserver?x      # URL query delimiter
smb_host:    docs%31          # percent-encoded host syntax is unsupported
smb_host:    문서서버          # non-ASCII/IDNA is unsupported in Milestone 1
smb_share:   docs/shared      # share name, not a path
smb_share:   docs#archive     # URL fragment delimiter
smb_share:   docs:archive     # UNC/path metacharacter
smb_share:   docs*archive     # UNC/path metacharacter
```

Server config rules:

- `id` must satisfy the `root_id` validation rules and be unique.
- `server_path` must be absolute after `filepath.Clean`.
- `name` is display metadata. Changing it must not require rebuilding the index.
- Changing a root `id` creates a new logical namespace and requires rebuilding
  the affected index documents.
- Changing `server_path` changes the indexed source of truth and requires
  re-indexing that root. If the old and new paths both exist, remove stale
  documents from the old root namespace during the same maintenance window.
- `smb_host` is the ASCII hostname, DNS name, Tailscale name, or IP address
  clients use to reach the Samba server.
- `smb_share` is the Samba share name, which may differ from the final
  `server_path` folder name.
- `smb_host` and `smb_share` are trimmed before validation and must remain
  non-empty after trimming.
- `smb_host` must be a host value only. It must not contain whitespace, control
  characters, non-ASCII text, `%`, `:`, `/`, `\`, `@`, `?`, or `#`, and must
  not include URL credentials or authority syntax. Milestone 1 supports only
  ASCII hostnames, IP addresses, and Tailscale names. IDNA, ports, and IPv6
  literals are unsupported until a later implementation explicitly normalizes
  them.
- `smb_share` must be a single share name, not a path. It must not contain
  whitespace, control characters, `/`, `\`, `?`, `#`, or UNC/path
  metacharacters (`:`, `*`, `"`, `<`, `>`, `|`).
- `smb_aliases` is optional per-root server metadata shaped as an array of
  `{ "host": "...", "share": "..." }` pairs. Aliases are operator-owned config
  because they authorize alternate shares, DFS names, or hostnames for the same
  root. Alias `host` and `share` values are trimmed, validated, and normalized
  with the same rules as `smb_host` and `smb_share`.
- Windows UNC paths are derived from `smb_host`, `smb_share`, and
  `relative_path` with an explicit Windows/UNC-semantics helper that is
  OS-independent and testable on macOS/Linux.
- macOS SMB URLs are derived from `smb_host`, `smb_share`, and `relative_path`;
  the desktop client still opens files through the mounted filesystem path.
- Overlapping roots are allowed. If multiple roots contain the same file, the
  most specific matching root owns that file.
- Two configured roots must not resolve to the exact same canonical
  `server_path`. Reject duplicate canonical roots during config validation.
- A parent root scan must skip subtrees owned by more specific child roots so
  the same physical file is not indexed under multiple logical IDs.
- `document_roots` is the only supported root configuration for this design.
  Existing `watched_paths` config is not migrated in place.
- `watched_paths` may remain in older local config files, but new SMB-index code
  must not silently convert it into document roots because root IDs and SMB share
  metadata are operator decisions.
- The committed `config.example.json` should move to this `document_roots`
  shape in the same implementation slice that introduces server-side
  `document_roots` loading.
- DocSearcher does not parse Samba configuration files. Operators must ensure
  that `smb_host` and `smb_share` expose the configured `server_path`. Optional
  diagnostics may test access to derived SMB paths, but config validation should
  not require Samba administrator access.
- Removing a root from config makes its indexed documents stale. The next
  maintenance window must remove documents for that `root_id` or rebuild the
  index, otherwise old search hits may point to roots no longer exposed by the
  root metadata API.

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
- Windows UNC overrides must normalize to the expected `smb_host` and
  `smb_share`, or match one normalized `smb_aliases` pair configured for that
  root. Reject wrong-host or wrong-share UNC overrides before containment
  checks.
- Windows drive-letter overrides cannot prove SMB host/share identity from the
  path alone. They require explicit user/operator configuration for that root
  and must still pass Windows containment checks after joining.
- Windows path parsing and containment must be implemented as OS-independent
  logic so tests can verify Windows drive and UNC behavior on non-Windows build
  hosts. Do not rely on the current build host's `filepath` semantics for
  Windows path rules.
- macOS clients use `smb://host/share` to identify or mount the share, then open
  files through local mounted paths such as `/Volumes/documents`.
- When deriving an SMB URL, escape `smb_share` as its own URL path segment, then
  escape each `relative_path` segment separately. Do not concatenate an
  unescaped share name with escaped file segments; share names may contain
  percent or non-ASCII characters that are valid after server validation.
- The client must join Windows drive or UNC roots with an explicit
  Windows-semantics helper, not the build host's path rules. It must join macOS
  local mount roots with OS path APIs. In both cases, reject any joined result
  that escapes the configured mount root after cleaning.
- Client-side open, reveal, and copy flows must revalidate raw `relative_path`
  values received from JSON APIs, HTMX `data-*` attributes, logical IDs, or
  cached DOM state before deriving UNC paths, SMB URLs, or mounted paths. The
  server remains the source of truth, but clients fail closed if a payload is
  hostile, stale, or no longer satisfies the same relative-path rules.
- Client-side `relative_path` validation must reject absolute paths, Windows
  drive prefixes, invalid UTF-8 text, control characters, `..` or `.`
  segments, `\`, `:`, Windows-illegal filename characters (`*`, `?`, `"`,
  `<`, `>`, `|`), reserved device basenames, empty segments such as
  `a//b.hwp`, and path segments ending in a space or `.`. This validation
  rejects malformed client payloads, including `a/./b.hwp`, instead of
  slash-normalizing them.
- macOS mount discovery and client-local mount overrides must verify that the
  chosen local mount root corresponds to the expected `smb_host` and
  `smb_share` when the OS exposes mount identity. If the identity is unavailable
  for verification or does not match the root metadata, fail with an explicit
  mount-unverified or mount-mismatch error instead of opening through a
  different share.
- Desktop clients must invoke OS open/reveal operations with argument-safe APIs
  or explicit argv lists. Do not build shell command strings from derived UNC,
  SMB URL, or mounted paths.

## API Contract

Root metadata is a first-class API contract, not an implementation detail.
`GET /api/document-roots` returns client-safe root metadata and a revision for
the root metadata set. The endpoint should be available to the browser UI and
desktop clients through the same server origin as search:

```json
{
  "revision": "roots-0007",
  "roots": [
    {
      "id": "documents",
      "name": "문서 공유",
      "smb_host": "docserver",
      "smb_share": "documents",
      "smb_aliases": [
        { "host": "dfs-docs", "share": "documents" }
      ]
    }
  ]
}
```

The root metadata API may expose only client-safe alias host/share pairs in
`smb_aliases`. It must not return `server_path`.

`revision` is server-generated metadata for the whole document-root metadata
set. It is opaque to clients and must change whenever any document root metadata
changes, including adding or removing a root, changing `id`, `name`,
`server_path`, `smb_host`, `smb_share`, or `smb_aliases`, or changing
validation-affecting canonicalization. Clients compare it only for equality as a
string or number and must not infer ordering, timestamps, or semantic meaning
from its value.

Clients may cache root metadata for responsiveness, but must track the
`revision`. Search responses and rendered HTMX fragments carry the root metadata
revision used for rendering. Before open/reveal, clients must refresh root
metadata when their cached revision is missing or differs from the search
result/fragment revision. They must also refresh when a search hit references an
unknown `root_id` or when an open/reveal attempt fails because root metadata is
missing. If open/reveal fails because the share is unavailable, resolution
fails, or the resolved file is missing, refresh root metadata once and retry
classification with fresh metadata before reporting the final error. A stale
cache must not cause fallback to a server-local path.

Search results should return logical file identity:

```json
{
  "id": "documents:shared/2026/sample.hwp",
  "root_id": "documents",
  "root_name": "문서 공유",
  "root_revision": "roots-0007",
  "relative_path": "shared/2026/sample.hwp",
  "fragment": "..."
}
```

Expected behavior:

- The browser UI displays the root name and relative path.
- The desktop client receives `root_id`, `root_revision`, and `relative_path`
  for open/reveal actions.
- The API must not treat Linux `server_path` values as client-openable paths.
- The desktop client reads `smb_host` and `smb_share` from the root metadata API
  by `root_id`.
- Search hits may include display metadata such as `root_name`, but they must
  remain logical and compact.
- Search hits and open payloads must not include `server_path`.
- Search hit `id` is the logical `document_id`. Clients should pass it around as
  an opaque identifier and use `root_id` plus `relative_path` for path
  resolution.
- Existing `/api/watch` UI/API behavior should be replaced by document-root
  management that edits or reports `document_roots`. New UI copy should use
  "document root" semantics rather than local watch-path semantics.
- Browser-only usage may show result metadata and copyable paths, but native
  open/reveal actions are available only through a desktop client bridge.
- Browser-only UI may show copy actions for `relative_path`, SMB URL, or UNC
  path when root metadata allows deriving them. It must derive SMB URLs with URL
  path escaping for the share segment and each relative-path segment, and derive
  UNC paths with the OS-independent Windows/UNC helper rather than string
  concatenation or the build host's path joining. It should not show native
  open/reveal buttons unless a desktop bridge is connected.
- When the current HTMX search flow renders HTML fragments, each result element
  should carry `data-document-id`, `data-root-id`, `data-root-revision`, and
  `data-relative-path` attributes for the desktop bridge. The fragment must not
  embed `server_path`.
- HTMX fragments must HTML-escape displayed paths and attribute values. A
  filename containing spaces, Korean characters, quotes, `<`, `>`, `&`, `#`, or
  `%` must not break markup, attributes, copy actions, or desktop bridge
  payloads.
- If a JSON search API is added for the Wails client, it should use the same
  logical fields as the example above and avoid a separate path contract.
- Desktop bridge calls should include an explicit capability version or feature
  flag for logical open payloads. The browser UI should enable native
  open/reveal only after that capability is observed.

## Open Flow

```text
1. User searches from desktop client.
2. Server returns root_id=documents and relative_path=shared/2026/sample.hwp.
3. Client loads root metadata for documents and refreshes it first if the
   cached revision is missing or differs from the search hit revision.
4. Client revalidates the raw `relative_path` payload and rejects hostile,
   malformed, or stale values, including `.` segments such as `a/./b.hwp`.
5. For macOS mount discovery or local mount overrides, the client verifies the
   selected mount root's exposed SMB identity against the expected
   `smb_host`/`smb_share` when the OS provides that identity, and fails with
   mount-unverified or mount-mismatch if it cannot prove the match.
6. For Windows overrides, UNC roots must match the expected normalized
   `smb_host`/`smb_share` or a normalized `smb_aliases` pair, while
   drive-letter roots require explicit root-level configuration and containment
   only.
7. Windows clients derive a UNC path from fresh `smb_host`, `smb_share`, and
   validated `relative_path`; macOS clients find or create a local mount from
   the SMB URL, then invoke OS-native file association/reveal APIs for the
   resulting path.
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

- Missing root metadata: tell the user the server did not return metadata for
  the search hit's `root_id`.
- Stale root metadata: refresh root metadata once, then report that the root is
  no longer available if the `root_id` is still missing.
- Missing SMB metadata: tell the user the root has no usable SMB host/share
  metadata.
- Missing local mount override: tell macOS users, or Windows users who selected
  a local override, that the client has no usable mount for that root.
- Missing file: tell the user the resolved SMB path does not exist or is not
  currently available.
- Permission failure: report that the OS denied access to the resolved file.
- Open failure: include enough resolved-path context for support without exposing
  unrelated local machine data.

The client should classify open errors before falling back to a generic failure.
For share availability, resolution, and missing-file failures, it should refresh
root metadata once, retry classification, and then stop; this rule must not loop
indefinitely.

1. Missing root metadata for `root_id`.
2. Stale root metadata after one refresh attempt.
3. Missing SMB share metadata or local mount mapping for the current platform.
4. Resolved path escapes the UNC/share/mount root after cleaning.
5. Resolved SMB path does not exist or the share is unavailable.
6. OS permission denial.
7. OS-native file association or reveal API failure.

Desktop clients may include the resolved UNC path, SMB URL, or local mounted
path in support-oriented errors. They must not include unrelated local paths,
server-only paths, credentials, or environment details.

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
- If a computed `relative_path` is not valid UTF-8 text or is not SMB-open-safe
  after validation, including any control character, backslash,
  Windows-illegal filename character, reserved Windows device basename, or any
  segment ending in a space or `.`, skip that file and log an operational
  warning. Do not encode, replace, rewrite, or silently normalize the path into
  a different logical document ID.
- Keep watcher-based indexing for normal changes.
- Expect Samba clients to save files through temp-file, rewrite, and rename
  patterns.
- Add a periodic rescan if watcher-only indexing proves unreliable under Samba
  saves.
- Watcher events must carry or recover root context so create/write/delete
  events compute the same logical `document_id` as the initial scan.
- Delete events should delete by logical `document_id`; they must not require
  reading a file that no longer exists.
- Directory delete events should delete all indexed documents for the affected
  `root_id` whose `relative_path` equals the deleted directory path or has that
  path as a slash-component prefix.
- Prefix matching must happen only after slash normalization. A deleted
  directory `a/b` matches `a/b/file.hwp`, but must not match `a/b2/file.hwp`.
- Directory rename events should remove indexed documents under the old
  `relative_path` prefix and schedule indexing for the new path when it remains
  under a configured root.
- If a watcher event cannot reliably distinguish file and directory state, or if
  event coalescing makes the old prefix uncertain, schedule a root
  reconciliation scan for the affected root.
- Root reconciliation compares the current scan result with indexed
  `root_id`/`relative_path` values and removes stale documents.
- Root reconciliation must also remove all indexed documents for configured
  roots that no longer exist in `document_roots`, unless the operator has chosen
  a full index rebuild instead.
- Store `root_id` and `relative_path` in Bleve. Store `server_path` only if it is
  needed for server-side diagnostics or re-index operations; it must not be
  returned as an openable client path.

## Bleve Storage Contract

Bleve document storage must make search result hydration independent from
implementation-specific ID parsing.

- Bleve document ID is the logical `document_id`.
- Store these fields for every indexed document:
  - `root_id`
  - `relative_path`
- Do not store `root_name` in Bleve as the display source of truth. Root names
  come from current root metadata at response-rendering time so renaming a root
  does not require index rebuild.
- Root IDs are stored as part of document identity. Renaming a root `id` is not a
  metadata-only change; rebuild or reconcile the affected documents under the new
  logical IDs.
- `server_path` may be stored only for server-side diagnostics or maintenance.
  It is a stored-only diagnostic field: never index it for search, never include
  it in queryable fields, never highlight it, and never return it to clients in
  search responses, root metadata responses, open payloads, fragments, or
  browser UI copy actions.
- Search hit hydration reads stored fields for `root_id` and `relative_path`.
  The hit ID may be returned as `id`, but it must not be the source of truth for
  path reconstruction.
- If stored fields are missing or invalid, treat the hit as corrupt index data:
  omit it from user-visible results, log enough server-side context to debug the
  issue, and repair it during the next full re-index or reconciliation scan.

## Cross-Platform Name Caveats

The Linux server may allow two different files whose names differ only by case,
while Windows or macOS SMB clients may present those names ambiguously depending
on share and filesystem settings.

- The first implementation does not deduplicate case-only filename collisions.
- Operators should avoid storing case-only sibling document names in shared
  roots that are opened from Windows or macOS clients.
- If a collision is detected during indexing or diagnostics, log it as an
  operational warning instead of silently rewriting logical IDs.
