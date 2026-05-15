# Client Migration

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

