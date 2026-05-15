# Server Index SMB Implementation Plan

## Expected Code Impact

This is a medium-size change because the current design assumes `DocumentID` is
the source path.

Likely affected areas:

- `internal/domain`: document identity, search hit fields, document root value
  types.
- `internal/infra/config`: persisted server document roots and Samba share
  metadata.
- `internal/usecase`: indexing path-to-document conversion and watch path
  handling.
- `internal/infra/search`: Bleve stored fields and search result hydration.
- `internal/server`: search result fragments and open payloads.
- `web/templates`: result click/double-click behavior.
- `cmd/client`: existing Windows WebView2 client behavior until the Wails client
  replaces it.
- Wails desktop client: SMB UNC/mount resolution, optional mount overrides,
  open/reveal behavior, and user-facing open errors.
- Tests for domain, config, indexing, search, server fragments, and client path
  joining.

Estimated implementation size:

```text
Central index model and server changes: 500-900 LOC
Windows SMB open UX:                   150-300 LOC
Tests and fixtures:                     300-600 LOC
Total first implementation:             800-1,500 LOC
```

## Cross-Platform Client Direction

Cross-platform desktop client work must target Wails. Treat the current
`cmd/client` Windows WebView2 entrypoint as the existing Windows client only
until the Wails replacement is introduced.

The Wails client owns:

- SMB open resolution from server-provided root metadata.
- optional local mount overrides by `root_id`.
- OS-native open and reveal actions.
- platform-specific path joining and availability checks.
- user-facing errors for missing mounts, missing files, and permission failures.

The Wails implementation should become the cross-platform desktop path for both
Windows and macOS instead of extending the Windows-only WebView2 client for
macOS support.

Migration from `cmd/client` to Wails can be phased, but new cross-platform client
features should be designed against the Wails client.

## Recommended First Milestone

Land the server contract before changing native open behavior:

1. Introduce validated `document_roots`, logical document IDs, and Bleve stored
   `root_id`/`relative_path` fields.
2. Update search responses and HTMX fragments to expose logical fields while
   keeping `server_path` out of client-visible payloads.
3. Add `GET /api/document-roots` with client-safe SMB metadata.
4. Reset and rebuild the index.

After that milestone, add Windows UNC open support and the Wails client work in
separate slices. If the existing `cmd/client` cannot safely consume the new
logical open payload during the first milestone, hide native open actions until
the Wails path is available.

Recommended Milestone 1 bridge policy: keep browser search usable, but hide or
disable native open/reveal actions unless the connected desktop bridge explicitly
advertises support for the logical `root_id` plus `relative_path` payload. Do
not pass server-local paths through the old `openFile(path)` bridge as a
compatibility shortcut.

Milestone 1 should also define the stale-root behavior: `GET
/api/document-roots` returns a root metadata `revision`, search
responses/fragments carry the revision used for rendering, and clients refresh
root metadata before open/reveal when their cached revision is missing or
differs. If a search hit refers to a `root_id` that is not returned by the
refreshed root metadata, clients show a root-unavailable error. They must not
infer a server path or reuse an old absolute-path bridge.

Milestone 1 `smb_host` validation should allow only ASCII hostnames, IP
addresses, and Tailscale names. Reject `%`, non-ASCII text, and `:`; IDNA,
ports, and IPv6 literals are unsupported until a later implementation explicitly
normalizes them for both UNC and SMB URL derivation.

## Milestone Scope

### Milestone 1: Server Logical Path Contract

Purpose: make the central index safe for SMB clients without promising native
desktop open behavior yet.

Included:

- `document_roots` config loading and validation.
- logical `document_id` creation from `root_id` and `relative_path`.
- most-specific-root ownership during scans.
- Bleve documents stored with logical IDs plus `root_id` and `relative_path`
  fields.
- search result hydration from stored logical fields.
- `GET /api/document-roots` returning client-safe SMB metadata.
- HTMX fragments carrying logical data attributes.
- browser UI display of root name and relative path.
- index reset and full re-index.

Excluded:

- automatic migration from `watched_paths`.
- native open/reveal behavior unless the active bridge already supports the
  logical payload.
- Wails packaging.
- Samba server configuration management.

Acceptance:

- Search and indexing work after rebuilding `hwp-index.bleve`.
- No search response, root metadata response, HTMX fragment, copy action, or
  open payload includes `server_path`.
- Existing browser search remains usable even when no desktop bridge is
  connected.
- Native open/reveal UI is hidden or disabled when the active bridge still
  expects local absolute paths.
- Legacy `watched_paths`-only config fails with an actionable operator message
  instead of being silently converted.
- Search hits for removed roots are absent after rebuild or reconciliation; if a
  stale hit appears during a transition, open/reveal fails closed.

### Milestone 2: Windows SMB Open

Purpose: let Windows users open indexed files from the central index.

Included:

- derive UNC paths from `smb_host`, `smb_share`, and `relative_path`.
- optional drive-letter or UNC mount overrides keyed by `root_id`.
- open and reveal actions through OS-native file association/reveal APIs.
- user-facing errors for missing root metadata, invalid mount overrides,
  unavailable files, permission failures, and OS open/reveal failures.

Acceptance:

- A Windows user can search, open, edit, save, and trigger server-side re-index
  of a document under the Samba share.
- UNC derivation and mount override containment are covered by OS-independent
  tests built around an explicit Windows/UNC-semantics helper so the rules are
  testable on macOS/Linux.

### Milestone 3: Wails Cross-Platform Client

Purpose: make the desktop client direction cross-platform and retire duplicated
open logic.

Included:

- Wails client consumes the logical search/open contract.
- Windows and macOS open/reveal flows.
- macOS SMB mount discovery or mount prompt flow.
- clear platform-specific errors.
- migration plan for retiring `cmd/client`.

Acceptance:

- Windows behavior remains at least as capable as Milestone 2.
- macOS users can open and reveal files through a mounted SMB share.
- `cmd/client` retirement is a separate cleanup after Wails reaches parity.

## Progress Checklist

### Completed Planning Decisions

- [x] Use logical document identity based on `root_id` and `relative_path`.
- [x] Keep Linux `server_path` as server-only operational state.
- [x] Return logical path fields to clients instead of openable Linux paths.
- [x] Use Wails as the cross-platform desktop client direction.
- [x] Treat `cmd/client` as the existing Windows WebView2 client until Wails
  replaces it.
- [x] Reset and rebuild the Bleve index instead of migrating existing documents
  in place.
- [x] Do not keep legacy `watched_paths` compatibility; require
  `document_roots`.
- [x] Support Windows UNC mounts as well as drive-letter mounts.
- [x] Store `root_id` and `relative_path` in Bleve; keep `server_path`
  server-only and optional.
- [x] Allow overlapping roots, with each file owned by the most specific
  matching root.
- [x] Use SMB/Samba as the primary native open backend.
- [x] Store Samba share metadata on server document roots; Windows opens use
  derived UNC paths by default.
- [x] Treat client mount mappings as optional overrides or macOS local mount
  state, not the primary open contract.
- [x] Use server/API logical path migration as Milestone 1.
- [x] Defer Wails packaging until after the server contract is test-covered.

### Implementation Tasks

Milestone sequencing:

- [ ] Land server config, logical IDs, search payloads, and root metadata API as
  the first implementation milestone.
- [ ] Gate or hide native open actions until the active desktop client can
  consume `root_id` and `relative_path`.
- [ ] Add Windows UNC open support after the server contract is stable.
- [ ] Add Wails macOS support after root metadata and open error contracts are
  test-covered.

Server contracts:

- [ ] Add domain types for document roots, relative paths, and logical document
  IDs.
- [ ] Add validation for `root_id`, `relative_path`, and logical document ID
  parsing.
- [ ] Reject `:` in `relative_path` values so logical IDs remain unambiguous and
  SMB-opened filenames stay Windows-compatible.
- [ ] Reject `\` in `relative_path` values because Windows UNC opens treat
  backslashes as separators.
- [ ] Reject Windows-illegal filename characters in `relative_path` segments:
  `*`, `?`, `"`, `<`, `>`, and `|`.
- [ ] Reject reserved Windows device basenames in `relative_path` segments:
  `CON`, `PRN`, `AUX`, `NUL`, `COM1`-`COM9`, and `LPT1`-`LPT9`,
  case-insensitive and including names with extensions.
- [ ] Reject `relative_path` segments ending in a space or `.` because Windows
  file APIs and UNC opens may trim or reject those names.
- [ ] Reject raw `relative_path` payloads containing `.` segments such as
  `a/./b.hwp`; do not clean or slash-normalize them into accepted paths.
- [ ] Reject `relative_path` values that are not valid UTF-8 text or contain
  control characters.
- [ ] Skip files whose computed `relative_path` is not valid UTF-8 text or is
  not SMB-open-safe and log an operational warning instead of encoding,
  replacing, rewriting, or silently normalizing the path.
- [ ] Document and test that logical IDs are stored as UTF-8 text and are not
  URL-encoded; URL escaping happens only when deriving SMB URLs.
- [ ] Treat root `id` changes as logical namespace changes that require index
  rebuild or root reconciliation.
- [ ] Add server-side `document_roots` config with `smb_host` and `smb_share`
  fields.
- [ ] Reject duplicate `document_roots[].id` values and `server_path` values
  that are relative or non-absolute after cleaning.
- [ ] Add validation for required SMB share metadata: trim `smb_host` and
  `smb_share`, reject empty values, slashes, backslashes, whitespace, and
  control characters; reject `%`, non-ASCII text, `:`, `@`, `?`, and `#` in
  `smb_host`; reject `:`, `*`, `?`, `"`, `<`, `>`, and `|` in `smb_share`; keep
  `smb_share` as a single share name without URL path/query/fragment
  delimiters. IDNA, ports, and IPv6 literals remain unsupported in Milestone 1.
- [ ] Add optional `smb_aliases` root metadata as an array of
  `{ "host": "...", "share": "..." }` pairs; validate and normalize alias
  values with the same rules as `smb_host` and `smb_share`.
- [ ] Treat `smb_aliases` as operator-owned server config because aliases
  authorize alternate shares, DFS names, or hostnames for a root.
- [ ] Canonicalize `server_path` roots with `EvalSymlinks`; reject unresolved
  root symlinks and keep the canonical root for containment checks.
- [ ] Reject duplicate canonical `server_path` roots during config validation,
  while still allowing parent/child overlaps that have different canonical
  roots.
- [ ] Implement path-component containment helpers for Unix paths instead of
  string-prefix checks.
- [ ] Implement OS-independent Windows drive and UNC parsing/containment helpers
  so Windows path behavior is testable on macOS/Linux.
- [ ] Remove or replace `watched_paths` config reads/writes in the server path.
- [ ] Update config UI/API language from watched paths to document roots.
- [ ] Update `config.example.json` from `watched_paths` to the
  `document_roots` shape when the new config loader lands.
- [ ] Rename stats/domain wording from watched-path count to document-root count.
- [ ] Define stale-root handling for search hits whose `root_id` is no longer in
  `document_roots`.

Indexing and search storage:

- [ ] Change indexing to compute relative paths from configured roots.
- [ ] Store `root_id`, `relative_path`, and optional diagnostic `server_path` in
  Bleve.
- [ ] Configure optional Bleve `server_path` as stored-only diagnostics: not
  indexed, not queryable, not highlighted, and never used for client hydration.
- [ ] Hydrate search results from stored `root_id` and `relative_path` fields
  rather than treating the Bleve hit ID as the path source of truth.
- [ ] Attach current root display names from root metadata during search response
  rendering instead of storing `root_name` as the index source of truth.
- [ ] Omit corrupt index hits with missing or invalid stored logical path fields
  from user-visible results and repair them during full re-index or
  reconciliation.
- [ ] Skip symlink file and directory entries during scans and watcher handling
  without following their targets.
- [ ] Ensure watcher create/write/delete events compute IDs from root context,
  not raw event paths.
- [ ] Delete file events by logical `document_id` without requiring the deleted
  file to still exist.
- [ ] Delete directory events by `root_id` plus slash-component
  `relative_path` prefix, so deleting `a/b` does not remove `a/b2`.
- [ ] Handle directory rename events by deleting old-prefix documents and
  scheduling indexing for the new prefix when it remains under a configured
  root.
- [ ] Handle overlapping roots by choosing the most specific matching root for
  each file.
- [ ] Skip more-specific child-root subtrees during parent-root scans to avoid
  duplicate logical documents.
- [ ] Re-index affected roots when root ownership changes, such as removing a
  child root that was excluding a subtree from its parent.
- [ ] Remove indexed documents for deleted root IDs during reconciliation or
  require a full index rebuild after root removal.
- [ ] When a root keeps the same `id` but changes `server_path`, remove stale
  documents from the old path during reconciliation or require a full rebuild.

Search API and web UI:

- [ ] Return search hits with logical path fields.
- [ ] Update HTMX search result fragments to carry `data-document-id`,
  `data-root-id`, `data-root-revision`, and `data-relative-path` for desktop
  bridge actions.
- [ ] HTML-escape displayed paths and `data-*` attributes so filenames with
  quotes, markup characters, Korean text, spaces, `#`, or `%` cannot break the
  fragment or bridge payload.
- [ ] Add a root metadata endpoint or equivalent payload for desktop clients to
  resolve `root_id` into SMB share metadata.
- [ ] Include a root metadata `revision` in `GET /api/document-roots`.
- [ ] Define `revision` as a server-generated opaque value for the whole root
  metadata set; change it whenever any root metadata changes, including
  add/remove, `id`, `name`, `server_path`, `smb_host`, `smb_share`, or
  `smb_aliases`, or validation-affecting canonicalization.
- [ ] Ensure root metadata responses expose only client-safe fields and never
  return `server_path`; if aliases are configured, expose only alias host/share
  pairs.
- [ ] Include the root metadata revision used for rendering in JSON search
  responses and HTMX fragments.
- [ ] Display root name and relative path instead of raw server absolute paths.
- [ ] Keep search responses and open payloads free of `server_path`.
- [ ] Add browser-only copy actions for logical path, SMB URL, or UNC path only
  when they can be derived without exposing server-local paths.
- [ ] Revalidate raw `relative_path` from API responses, HTMX `data-*`
  attributes, cached DOM state, or logical IDs before browser copy derives SMB
  URLs or UNC paths; reject hostile or stale payloads with absolute paths,
  Windows drive prefixes, invalid UTF-8 text, control characters, `..`, `.`,
  `\`, `:`, Windows-illegal characters, reserved device basenames, empty
  segments such as `a//b.hwp`, or trailing space/dot segments. Do not
  slash-normalize malformed client payloads such as `a/./b.hwp`.
- [ ] Derive SMB URLs by URL-escaping `smb_share` as one path segment and each
  `relative_path` segment separately.
- [ ] Keep browser-only result actions separate from desktop-client open/reveal
  actions.
- [ ] Add an explicit desktop bridge capability check for logical open/reveal
  payloads before showing native actions.
- [ ] Update README, `docs/contracts.md`, and `ARCHITECTURE.md` references that
  still describe `watched_paths` as the active configuration contract.

Wails desktop client:

- [ ] Let Wails coexist with `cmd/client` until Windows behavior is
  feature-complete, then retire `cmd/client` in a separate cleanup.
- [ ] Resolve Windows open paths from `smb_host`, `smb_share`, and
  `relative_path` into UNC paths.
- [ ] Support drive-letter mounts as optional Windows overrides.
- [ ] Require Windows UNC overrides to match the expected normalized
  `smb_host`/`smb_share` or any normalized `smb_aliases` pair configured for
  that root; reject wrong-host or wrong-share overrides.
- [ ] Require explicit user/operator configuration for Windows drive-letter
  overrides because the path alone cannot prove host/share identity; still
  enforce containment after joining.
- [ ] Derive SMB URLs with URL path escaping, and derive UNC paths without URL
  escaping by using the OS-independent Windows/UNC helper.
- [ ] Refresh root metadata before desktop open/reveal when the cached revision
  is missing or differs from the search result/fragment revision.
- [ ] Revalidate raw `relative_path` payloads before desktop open/reveal path
  derivation so malicious or stale client data fails closed before UNC, SMB URL,
  or mount path derivation.
- [ ] Invoke OS open/reveal through argument-safe APIs or argv lists, never by
  concatenating derived paths into shell command strings.
- [ ] Resolve macOS open paths by finding or mounting `smb://host/share`, then
  joining the local mounted path with `relative_path`.
- [ ] Verify macOS-discovered mounts and client-local mount overrides match the
  expected `smb_host` and `smb_share` when the OS exposes mount identity.
- [ ] Fail macOS open/reveal with explicit mount-unverified or mount-mismatch
  errors when mount identity cannot be verified or does not match, instead of
  opening from the wrong share.
- [ ] Reject path joins that escape the configured mount root.
- [ ] Open files through OS-native file association/default app behavior.
- [ ] Add "show in folder" behavior.
- [ ] Add clear open failures for missing root metadata, missing SMB metadata,
  missing local mount overrides, unavailable files, denied permissions, and
  OS open/reveal failures.
- [ ] On share availability, resolution, or missing-file failures, refresh root
  metadata once, retry open/reveal error classification with fresh metadata, and
  then report the final error without further retries.

Client migration:

- [ ] Keep Milestone 1 server/API/logical payload only; defer Windows UNC open
  and Wails work until after the server contract is stable.
- [ ] If `cmd/client` remains during the first milestone, keep its open payload
  compatible with the new logical `root_id`/`relative_path` bridge contract or
  hide native open actions until Wails is available.
- [ ] Do not reuse the legacy `openFile(path)` bridge with Linux `server_path`
  values during the SMB migration.
- [ ] Refresh root metadata once when opening a hit with an unknown `root_id`,
  then fail with a root-unavailable error if it remains unknown.
- [ ] Refresh root metadata once when cached SMB metadata leads to share
  availability, path resolution, or missing-file failures, then retry
  classification before showing the final error.

Index reliability:

- [ ] Add root reconciliation scans for watcher events that cannot reliably
  identify file/directory state or old directory prefixes.
- [ ] Compare reconciliation scan results with indexed `root_id`/`relative_path`
  values and remove stale documents.
- [ ] Add periodic rescan only if watcher testing under Samba shows missed
  events.
- [ ] Log operational warnings for case-only sibling filename collisions under a
  root because Windows/macOS SMB clients may present them ambiguously.

## Migration Approach

Do not migrate existing Bleve documents in place. The current index stores
absolute paths as IDs, so the simpler and safer migration is:

1. Stop the DocSearcher server.
2. Back up `config.json` and record the old watched paths for operator
   reference.
3. Add explicit `document_roots` config.
4. Change indexing to write logical IDs.
5. Back up, archive, or remove the old `hwp-index.bleve`.
6. Start the server and re-index configured document roots.
7. Confirm search results, fragments, and root metadata expose logical fields
   only.

No `watched_paths` compatibility layer is planned. Operators should replace
existing local config with explicit `document_roots` that include
operator-defined `id`, `server_path`, `smb_host`, and `smb_share`.

Migration rules:

- If `config.json` contains only `watched_paths`, treat it as an old config and
  require operators to write `document_roots` before using the SMB design.
- If both `document_roots` and `watched_paths` exist, ignore `watched_paths`.
- Do not generate root IDs from legacy paths.
- Rebuild `hwp-index.bleve` whenever changing from absolute-path IDs to logical
  IDs.
- Rebuild or reconcile affected documents whenever a root `id` or `server_path`
  changes.
- Rebuild or remove stale documents whenever a root is deleted from
  `document_roots`.

## Closed Follow-Up Questions

- UI error policy is closed for Milestone 1: missing root metadata, missing SMB
  metadata, and missing mount overrides should produce explicit user-facing
  errors instead of path guessing. Later work may refine copy and localization.
- Root metadata cache policy is closed for Milestone 1: `GET
  /api/document-roots` returns a root metadata `revision`; search
  responses/fragments carry the revision used for rendering; clients refresh
  root metadata before open/reveal when their cached revision is missing or
  differs, and still refresh once on unknown `root_id`, missing-metadata open
  failure, or cached-SMB resolution, availability, or missing-file failure.

## Remaining Open Questions

- Should server diagnostics expose `server_path` to admins through a protected
  endpoint, or only logs?
