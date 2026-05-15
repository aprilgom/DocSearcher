# Implementation Overview

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

# Cross-Platform Client Direction

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

# Recommended First Milestone

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

# Milestone Scope

## Milestone 1: Server Logical Path Contract

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

## Milestone 2: Windows SMB Open

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

## Milestone 3: Wails Cross-Platform Client

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

