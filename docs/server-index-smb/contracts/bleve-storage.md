# Bleve Storage Contract

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

