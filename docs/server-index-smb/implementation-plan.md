# Server Index SMB Implementation Plan

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
- `cmd/client`: existing Windows WebView2 client behavior until the Wails client
  replaces it.
- Wails desktop client: cross-platform mount mapping, path resolution,
  open/reveal behavior, and user-facing open errors.
- Tests for domain, config, indexing, search, server fragments, and client path
  joining.

Estimated implementation size:

```text
Central index model and server changes: 500-900 LOC
Windows client mount mapping/open UX:   150-300 LOC
Tests and fixtures:                     300-600 LOC
Total first implementation:             800-1,500 LOC
```

## Cross-Platform Client Direction

Cross-platform desktop client work must target Wails. Treat the current
`cmd/client` Windows WebView2 entrypoint as the existing Windows client only
until the Wails replacement is introduced.

The Wails client owns:

- local mount configuration by `root_id`.
- OS-native open and reveal actions.
- platform-specific path joining and availability checks.
- user-facing errors for missing mounts, missing files, and permission failures.

The Wails implementation should become the cross-platform desktop path for both
Windows and macOS instead of extending the Windows-only WebView2 client for
macOS support.

Migration from `cmd/client` to Wails can be phased, but new cross-platform client
features should be designed against the Wails client.

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
- [x] Accept existing `watched_paths` for one transition release, but make
  `document_roots` the new config contract.
- [x] Support Windows UNC mounts as well as drive-letter mounts.
- [x] Store `root_id` and `relative_path` in Bleve; keep `server_path`
  server-only and optional.
- [x] Allow overlapping roots, with each file owned by the most specific
  matching root.

### Implementation Tasks

Server contracts:

- [ ] Add domain types for document roots, relative paths, and logical document
  IDs.
- [ ] Add validation for `root_id`, `relative_path`, and logical document ID
  parsing.
- [ ] Add server-side `document_roots` config.
- [ ] Read legacy `watched_paths` as transition input and normalize them into
  generated document roots where possible.
- [ ] Keep current watched-path behavior only where needed for compatibility,
  while writing new configuration as `document_roots`.

Indexing and search storage:

- [ ] Change indexing to compute relative paths from configured roots.
- [ ] Store `root_id`, `relative_path`, and optional `server_path` in Bleve.
- [ ] Ensure watcher create/write/delete events compute IDs from root context,
  not raw event paths.
- [ ] Handle overlapping roots by choosing the most specific matching root for
  each file.
- [ ] Skip more-specific child-root subtrees during parent-root scans to avoid
  duplicate logical documents.
- [ ] Re-index affected roots when root ownership changes, such as removing a
  child root that was excluding a subtree from its parent.

Search API and web UI:

- [ ] Return search hits with logical path fields.
- [ ] Update web result actions to pass `root_id` and `relative_path`.
- [ ] Display root name and relative path instead of raw server absolute paths.
- [ ] Keep browser-only result actions separate from desktop-client open/reveal
  actions.

Wails desktop client:

- [ ] Let Wails coexist with `cmd/client` until Windows behavior is
  feature-complete, then retire `cmd/client` in a separate cleanup.
- [ ] Add Wails client support for resolving root mounts into local SMB paths.
- [ ] Support Windows drive-letter mounts and UNC mounts.
- [ ] Reject path joins that escape the configured mount root.
- [ ] Open files through OS-native shell/default app behavior.
- [ ] Add "show in folder" behavior.
- [ ] Add clear open failures for missing mounts, unavailable files, and denied
  permissions.

Index reliability:

- [ ] Add periodic rescan only if watcher testing under Samba shows missed
  events.

## Migration Approach

Do not migrate existing Bleve documents in place. The current index stores
absolute paths as IDs, so the simpler and safer migration is:

1. Add document root config.
2. Change indexing to write logical IDs.
3. Reset `hwp-index.bleve`.
4. Re-index configured document roots.

During a transition, existing `watched_paths` config can be treated as legacy
input. New deployments should use `document_roots`.

Transition rules:

- If only `watched_paths` exists, load each path as a generated document root
  with a stable derived ID, then prompt operators to save explicit
  `document_roots`.
- If both `document_roots` and `watched_paths` exist, prefer
  `document_roots`.
- After one release, remove writes to `watched_paths`; later removal of read
  compatibility can be handled as a separate cleanup.
- Rebuild `hwp-index.bleve` whenever changing from absolute-path IDs to logical
  IDs.

## Remaining Open Questions

- What naming convention should generated root IDs use for legacy
  `watched_paths` when no operator-supplied ID exists?
- Should server diagnostics expose `server_path` to admins through a protected
  endpoint, or only logs?
- How should the UI guide desktop users to create a missing mount mapping for a
  returned `root_id`?
