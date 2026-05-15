# Indexing And Watch Rules

The indexer scans each configured server root directly:

```text
root_id=documents
server_path=/data/documents
```

For every supported file under `server_path`, compute:

```text
relative_path = filepath.Rel(server_path, file_path)
document_id   = root_id + ":" + slash-normalized relative_path
```

Root ownership rules:

- A file is owned by the most specific configured root that contains it.
- Initial scans, watcher create/write/delete events, and stale-index cleanup
  must all use the same ownership rule.
- If a parent root is scanned, directories owned by more specific child roots
  are skipped during that parent scan. The child root scan is responsible for
  indexing those files.
- Adding a parent root around an existing child root is allowed, but the parent
  root indexes only the files not owned by the child root.
- Removing a child root can move ownership of its files to a parent root; handle
  that as an explicit root-configuration change that requires re-indexing the
  affected roots.

Implementation requirements:

- Keep temporary and lock files excluded by the supported document path policy.
- If a computed `relative_path` is not valid UTF-8 text or is not SMB-open-safe
  after validation, including any control character, backslash,
  Windows-illegal filename character, reserved Windows device basename, or any
  segment ending in a space or `.`, skip that file and log an operational
  warning. Do not encode, replace, rewrite, or silently normalize the path into
  a different logical document ID.
- Keep watcher-based indexing for normal changes.
- Expect Samba clients to save files through temp-file, rewrite, and rename
  patterns.
- Add a periodic rescan if watcher-only indexing proves unreliable under Samba
  saves.
- Watcher events must carry or recover root context so create/write/delete
  events compute the same logical `document_id` as the initial scan.
- Delete events should delete by logical `document_id`; they must not require
  reading a file that no longer exists.
- Directory delete events should delete all indexed documents for the affected
  `root_id` whose `relative_path` equals the deleted directory path or has that
  path as a slash-component prefix.
- Prefix matching must happen only after slash normalization. A deleted
  directory `a/b` matches `a/b/file.hwp`, but must not match `a/b2/file.hwp`.
- Directory rename events should remove indexed documents under the old
  `relative_path` prefix and schedule indexing for the new path when it remains
  under a configured root.
- If a watcher event cannot reliably distinguish file and directory state, or if
  event coalescing makes the old prefix uncertain, schedule a root
  reconciliation scan for the affected root.
- Root reconciliation compares the current scan result with indexed
  `root_id`/`relative_path` values and removes stale documents.
- Root reconciliation must also remove all indexed documents for configured
  roots that no longer exist in `document_roots`, unless the operator has chosen
  a full index rebuild instead.
- Store `root_id` and `relative_path` in Bleve. Store `server_path` only if it is
  needed for server-side diagnostics or re-index operations; it must not be
  returned as an openable client path.

