# Completed Planning Decisions

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

# Implementation Tasks

