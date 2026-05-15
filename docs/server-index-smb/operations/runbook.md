# Deployment Runbook

Use this sequence when moving an existing local-path deployment to the central
SMB-index design:

1. Stop the DocSearcher server.
2. Back up `config.json` and any operational notes needed to recreate the
   current watched paths.
3. Replace `watched_paths` with explicit `document_roots`, including stable
   `id`, display `name`, Linux `server_path`, `smb_host`, and `smb_share`.
4. Verify the Samba share exposes the same directory as `server_path`.
5. Remove or archive the old `hwp-index.bleve` directory.
6. Start the server and allow a full re-index.
7. Search from the browser UI and confirm results show root name plus relative
   path only.
8. Confirm `GET /api/document-roots` returns a root metadata `revision` and
   does not expose `server_path`; configured aliases appear only as host/share
   pairs.
9. Confirm no search result refers to a removed or renamed root ID after the
   rebuild or reconciliation completes.
10. Confirm search responses/fragments carry the root metadata revision used for
   rendering.
11. Enable desktop open/reveal only after the active desktop client supports the
   logical `root_id` plus `relative_path` payload and refreshes stale or missing
   root metadata revisions before open/reveal.

Rollback is a restore operation, not an index migration. Stop the server,
restore the previous config and index backup if one was kept, then restart. Do
not try to mix absolute-path index documents with logical-ID documents in the
same Bleve index.

The Bleve index is rebuildable operational data. Back it up only when it helps
rollback speed or diagnostics. The source documents under the Samba share are
not rebuildable from the index and need their own backup or snapshot policy.

