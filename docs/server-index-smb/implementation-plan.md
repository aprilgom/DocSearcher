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

## Implementation Checklist

1. Add domain types for document roots, relative paths, and logical document IDs.
2. Add server-side `document_roots` config.
3. Keep current watched-path behavior only where needed for compatibility.
4. Change indexing to compute relative paths from configured roots.
5. Store `root_id`, `relative_path`, and optional `server_path` in Bleve.
6. Return search hits with logical path fields.
7. Update web result actions to pass `root_id` and `relative_path`.
8. Add Wails client support for resolving root mounts into local SMB paths.
9. Open files through OS-native shell/default app behavior.
10. Add "show in folder" behavior.
11. Add clear open failures for missing mounts, unavailable files, and denied
    permissions.
12. Decide whether the Wails client replaces `cmd/client` in one migration or
    coexists until Windows behavior is feature-complete.
13. Add periodic rescan only if watcher testing under Samba shows missed events.

## Migration Approach

Do not migrate existing Bleve documents in place. The current index stores
absolute paths as IDs, so the simpler and safer migration is:

1. Add document root config.
2. Change indexing to write logical IDs.
3. Reset `hwp-index.bleve`.
4. Re-index configured document roots.

During a transition, existing `watched_paths` config can be treated as legacy
input. New deployments should use `document_roots`.

## Open Questions

- Should `document_roots` fully replace `watched_paths`, or should both be
  accepted for one release?
- Should the server store `server_path` in Bleve, or reconstruct it from
  `root_id` and `relative_path` when needed?
- Should client mounts accept UNC paths like `\\docserver\documents` in addition
  to drive-letter paths?
- How should the UI expose mount setup when a desktop client has no mapping for a
  returned `root_id`?
