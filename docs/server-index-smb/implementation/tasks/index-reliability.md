# Index Reliability

- [ ] Add root reconciliation scans for watcher events that cannot reliably
  identify file/directory state or old directory prefixes.
- [ ] Compare reconciliation scan results with indexed `root_id`/`relative_path`
  values and remove stale documents.
- [ ] Add periodic rescan only if watcher testing under Samba shows missed
  events.
- [ ] Log operational warnings for case-only sibling filename collisions under a
  root because Windows/macOS SMB clients may present them ambiguously.

