# Data Model

Use a stable logical document identity instead of client-specific or
server-specific absolute paths.

Current local-machine model:

```text
document_id = /data/documents/shared/2026/sample.hwp
path        = /data/documents/shared/2026/sample.hwp
```

Target central-index model:

```text
root_id       = documents
relative_path = shared/2026/sample.hwp
document_id   = documents:shared/2026/sample.hwp
server_path   = /data/documents/shared/2026/sample.hwp
smb_unc       = \\docserver\documents
smb_url       = smb://docserver/documents
```

Rules:

- `root_id` identifies a configured server document root.
- `relative_path` is computed from the root's `server_path`.
- `relative_path` is slash-normalized for storage and API contracts.
- `document_id` is `root_id + ":" + relative_path`.
- `server_path` is server-only operational state for parsing, watching, and
  re-indexing.
- SMB is the primary open backend. Each server document root should describe the
  Samba share that exposes the same root to desktop clients.
- Search results expose `root_id` and `relative_path`, not openable Linux paths.

Validation rules:

- `root_id` must be stable ASCII using lowercase letters, numbers, `_`, and
  `-`. It must not contain `:`, `/`, `\`, whitespace, or empty segments.
- `relative_path` must be non-empty, slash-normalized, and relative to the
  selected configured root. Slash normalization is only for controlled server
  filesystem-to-relative-path computation.
- `relative_path` must be valid UTF-8 text and must not contain control
  characters. Invalid UTF-8 and control characters are rejected; they are not
  encoded, replaced, or normalized into another name.
- Raw API, DOM, or logical-ID payloads for `relative_path` must not be absolute,
  contain Windows drive prefixes, contain `..` or `.` segments, contain `:`,
  contain `\`, contain Windows-illegal filename characters (`*`, `?`, `"`,
  `<`, `>`, `|`), contain empty path segments such as `a//b.hwp`, or contain
  any segment ending in a space or `.`. Reject malformed client payloads before
  deriving paths; do not normalize them into an acceptable value. In
  particular, reject `a/./b.hwp` as malformed rather than cleaning it to
  `a/b.hwp`. This keeps logical IDs unambiguous and avoids
  Windows-incompatible names in the SMB open path.
  Backslashes are rejected because Windows UNC APIs treat them as path
  separators, so accepting them as literal filename text would change the
  opened path. Trailing segment spaces and dots are rejected because Windows
  file APIs and UNC opens may trim or reject those names.
- Each `relative_path` segment must not have a reserved Windows device basename:
  `CON`, `PRN`, `AUX`, `NUL`, `COM1`-`COM9`, or `LPT1`-`LPT9`, case-insensitive
  and including names with extensions such as `CON.txt`.
- `document_id` is built only after `root_id` and `relative_path` validation.
  Code that needs to split a `document_id` must split on the first `:` and then
  validate both parts.
- `document_id` is not URL-encoded. Store it as `root_id`, a literal `:`, and
  the slash-normalized UTF-8 `relative_path`. Apply URL escaping only when
  deriving SMB URLs for copy or mount flows.
- Do not accept client-supplied `server_path` values. Server paths are derived
  from trusted server config plus validated relative paths.
- Clients must not reconstruct `server_path` from logical fields or submit a
  guessed server path back to the server for open, reveal, delete, or re-index
  actions.

Path containment rules:

- Never use string-prefix checks to decide whether a file is contained by a
  root. Containment is path-component based.
- On Unix-like platforms, clean both paths, compute `filepath.Rel(root, file)`,
  and accept only non-absolute results where `rel` is neither `..` nor prefixed
  by `../`. Names such as `..draft/file.hwp` are valid contained paths when the
  component check passes.
- The root itself must resolve successfully during config validation. If
  `server_path` is a symlink, resolve it with `EvalSymlinks` and use that
  canonical root for containment and relative-path computation. A root symlink
  that cannot be resolved is a config error.
- Files that disappear during scan or watch handling may be skipped and retried
  by a later event or reconciliation scan.
- Symlink entries are not followed and are not indexed during scans or watcher
  handling. This avoids duplicate logical documents for the same physical file
  and prevents symlink escapes outside the configured root.
- A symlink entry is skipped, not treated as a fatal root error, because a
  single user-created or tool-created link must not stop indexing the rest of
  the root.
- On Windows, containment must compare path components case-insensitively after
  cleaning. Drive roots or UNC share roots must match before relative path
  comparison.
- Windows drive and UNC prefix traps must be rejected: `Z:\docs2` is not under
  `Z:\docs`, and `\\server\share2` is not under `\\server\share`.

