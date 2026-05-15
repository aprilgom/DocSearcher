# Server Contracts

- [ ] Add domain types for document roots, relative paths, and logical document
  IDs.
- [ ] Add validation for `root_id`, `relative_path`, and logical document ID
  parsing.
- [ ] Reject `:` in `relative_path` values so logical IDs remain unambiguous and
  SMB-opened filenames stay Windows-compatible.
- [ ] Reject `\` in `relative_path` values because Windows UNC opens treat
  backslashes as separators.
- [ ] Reject Windows-illegal filename characters in `relative_path` segments:
  `*`, `?`, `"`, `<`, `>`, and `|`.
- [ ] Reject reserved Windows device basenames in `relative_path` segments:
  `CON`, `PRN`, `AUX`, `NUL`, `COM1`-`COM9`, and `LPT1`-`LPT9`,
  case-insensitive and including names with extensions.
- [ ] Reject `relative_path` segments ending in a space or `.` because Windows
  file APIs and UNC opens may trim or reject those names.
- [ ] Reject raw `relative_path` payloads containing `.` segments such as
  `a/./b.hwp`; do not clean or slash-normalize them into accepted paths.
- [ ] Reject `relative_path` values that are not valid UTF-8 text or contain
  control characters.
- [ ] Skip files whose computed `relative_path` is not valid UTF-8 text or is
  not SMB-open-safe and log an operational warning instead of encoding,
  replacing, rewriting, or silently normalizing the path.
- [ ] Document and test that logical IDs are stored as UTF-8 text and are not
  URL-encoded; URL escaping happens only when deriving SMB URLs.
- [ ] Treat root `id` changes as logical namespace changes that require index
  rebuild or root reconciliation.
- [ ] Add server-side `document_roots` config with `smb_host` and `smb_share`
  fields.
- [ ] Reject duplicate `document_roots[].id` values and `server_path` values
  that are relative or non-absolute after cleaning.
- [ ] Add validation for required SMB share metadata: trim `smb_host` and
  `smb_share`, reject empty values, slashes, backslashes, whitespace, and
  control characters; reject `%`, non-ASCII text, `:`, `@`, `?`, and `#` in
  `smb_host`; reject `:`, `*`, `?`, `"`, `<`, `>`, and `|` in `smb_share`; keep
  `smb_share` as a single share name without URL path/query/fragment
  delimiters. IDNA, ports, and IPv6 literals remain unsupported in Milestone 1.
- [ ] Add optional `smb_aliases` root metadata as an array of
  `{ "host": "...", "share": "..." }` pairs; validate and normalize alias
  values with the same rules as `smb_host` and `smb_share`.
- [ ] Treat `smb_aliases` as operator-owned server config because aliases
  authorize alternate shares, DFS names, or hostnames for a root.
- [ ] Canonicalize `server_path` roots with `EvalSymlinks`; reject unresolved
  root symlinks and keep the canonical root for containment checks.
- [ ] Reject duplicate canonical `server_path` roots during config validation,
  while still allowing parent/child overlaps that have different canonical
  roots.
- [ ] Implement path-component containment helpers for Unix paths instead of
  string-prefix checks.
- [ ] Implement OS-independent Windows drive and UNC parsing/containment helpers
  so Windows path behavior is testable on macOS/Linux.
- [ ] Remove or replace `watched_paths` config reads/writes in the server path.
- [ ] Update config UI/API language from watched paths to document roots.
- [ ] Update `config.example.json` from `watched_paths` to the
  `document_roots` shape when the new config loader lands.
- [ ] Rename stats/domain wording from watched-path count to document-root count.
- [ ] Define stale-root handling for search hits whose `root_id` is no longer in
  `document_roots`.

