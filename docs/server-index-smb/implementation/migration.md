# Migration Approach

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

# Closed Follow-Up Questions

- UI error policy is closed for Milestone 1: missing root metadata, missing SMB
  metadata, and missing mount overrides should produce explicit user-facing
  errors instead of path guessing. Later work may refine copy and localization.
- Root metadata cache policy is closed for Milestone 1: `GET
  /api/document-roots` returns a root metadata `revision`; search
  responses/fragments carry the revision used for rendering; clients refresh
  root metadata before open/reveal when their cached revision is missing or
  differs, and still refresh once on unknown `root_id`, missing-metadata open
  failure, or cached-SMB resolution, availability, or missing-file failure.

# Remaining Open Questions

- Should server diagnostics expose `server_path` to admins through a protected
  endpoint, or only logs?
