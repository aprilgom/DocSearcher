# Indexing And Search Storage

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

