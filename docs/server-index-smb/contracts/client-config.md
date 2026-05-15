# Client Config

SMB share metadata in `document_roots` is the primary source for open
resolution. Client-side mount config is an override or local state, not the main
contract.

Windows clients should open search hits through the derived UNC path by default:

```text
\\docserver\documents\shared\2026\sample.hwp
```

Client-side runtime config may map server root IDs to local SMB mount locations
when a machine needs a local override:

```json
{
  "mounts": {
    "documents": "Z:\\"
  }
}
```

macOS example:

```json
{
  "mounts": {
    "documents": "/Volumes/documents"
  }
}
```

`mounts` is local machine state because drive letters and mount paths can differ
per user and operating system.

Client config rules:

- A mount key must match a server `root_id`.
- Windows clients should prefer the server-derived UNC path unless the user
  explicitly chooses a drive-letter or local override.
- Windows mounts may be drive-letter roots such as `Z:\` or UNC roots such as
  `\\docserver\documents`.
- Windows UNC overrides must normalize to the expected `smb_host` and
  `smb_share`, or match one normalized `smb_aliases` pair configured for that
  root. Reject wrong-host or wrong-share UNC overrides before containment
  checks.
- Windows drive-letter overrides cannot prove SMB host/share identity from the
  path alone. They require explicit user/operator configuration for that root
  and must still pass Windows containment checks after joining.
- Windows path parsing and containment must be implemented as OS-independent
  logic so tests can verify Windows drive and UNC behavior on non-Windows build
  hosts. Do not rely on the current build host's `filepath` semantics for
  Windows path rules.
- macOS clients use `smb://host/share` to identify or mount the share, then open
  files through local mounted paths such as `/Volumes/documents`.
- When deriving an SMB URL, escape `smb_share` as its own URL path segment, then
  escape each `relative_path` segment separately. Do not concatenate an
  unescaped share name with escaped file segments; share names may contain
  percent or non-ASCII characters that are valid after server validation.
- The client must join Windows drive or UNC roots with an explicit
  Windows-semantics helper, not the build host's path rules. It must join macOS
  local mount roots with OS path APIs. In both cases, reject any joined result
  that escapes the configured mount root after cleaning.
- Client-side open, reveal, and copy flows must revalidate raw `relative_path`
  values received from JSON APIs, HTMX `data-*` attributes, logical IDs, or
  cached DOM state before deriving UNC paths, SMB URLs, or mounted paths. The
  server remains the source of truth, but clients fail closed if a payload is
  hostile, stale, or no longer satisfies the same relative-path rules.
- Client-side `relative_path` validation must reject absolute paths, Windows
  drive prefixes, invalid UTF-8 text, control characters, `..` or `.`
  segments, `\`, `:`, Windows-illegal filename characters (`*`, `?`, `"`,
  `<`, `>`, `|`), reserved device basenames, empty segments such as
  `a//b.hwp`, and path segments ending in a space or `.`. This validation
  rejects malformed client payloads, including `a/./b.hwp`, instead of
  slash-normalizing them.
- macOS mount discovery and client-local mount overrides must verify that the
  chosen local mount root corresponds to the expected `smb_host` and
  `smb_share` when the OS exposes mount identity. If the identity is unavailable
  for verification or does not match the root metadata, fail with an explicit
  mount-unverified or mount-mismatch error instead of opening through a
  different share.
- Desktop clients must invoke OS open/reveal operations with argument-safe APIs
  or explicit argv lists. Do not build shell command strings from derived UNC,
  SMB URL, or mounted paths.

