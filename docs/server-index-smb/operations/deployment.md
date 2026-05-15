# Target Deployment

```text
Linux document server
  /data/documents
    Source HWP/HWPX/PDF files
  hwp-index.bleve/
    Single server-side Bleve index
  Samba share
    /data/documents -> \\docserver\documents

Windows client
  Opens the derived UNC path \\docserver\documents\...
  May optionally use a drive-letter override such as Z:\
  Opens search hits through OS-native file association/reveal APIs

macOS client
  Locates or mounts the same share as smb://docserver/documents
  Opens search hits through /Volumes/documents or another mounted path

Tailscale
  Provides private network reachability to the Linux server and SMB port
```

Clients do not build local indexes. The Linux server owns document storage,
filesystem watching, parsing, and indexing.

The DocSearcher server root and Samba share should be configured together:

```text
document_root.id       = documents
document_root.server_path = /data/documents
document_root.smb_host = docserver
document_root.smb_share = documents
document_root.smb_aliases = [{ host = dfs-docs, share = documents }]
```

The Samba share name does not have to match the final folder name, but it must
identify the share that exposes the configured `server_path`.

`smb_host` and `smb_share` must be plain UNC/SMB URL components. Trim both
values, reject empty values, reject slashes, backslashes, whitespace, and control
characters, reject `%`, non-ASCII text, and `:` in `smb_host`, reject UNC/path
metacharacters in `smb_share` (`:`, `*`, `?`, `"`, `<`, `>`, `|`), and do not
put IDNA names, ports, IPv6 literals, credentials, paths, query strings, or
fragments in either field. Milestone 1 `smb_host` values are ASCII-only
hostnames, IP addresses, or Tailscale names.

`smb_aliases` is optional per-root server config shaped as host/share pairs.
Aliases are operator-owned because they authorize alternate shares, DFS names,
or hostnames for the same indexed root. Alias values follow the same trimming,
validation, and normalization rules as `smb_host` and `smb_share`.

DocSearcher treats this mapping as an operator-owned deployment contract. It can
derive and display SMB paths from `smb_host`, `smb_share`, and `relative_path`,
but it should not require permission to read Samba server configuration files.
Files whose `relative_path` is not valid UTF-8 text or is not SMB-open-safe,
including control characters, Windows-illegal filename characters, reserved
device basenames, or path segments ending in a space or `.`, should be skipped
with operational warnings instead of encoded, replaced, rewritten, or silently
normalized.

Desktop bridge and browser copy code must still revalidate raw `relative_path`
payloads received from API responses, HTMX fragments, DOM state, or logical IDs
before deriving UNC, SMB URL, or mounted paths. This client-side check is
defense in depth for stale or hostile payloads; it must reject absolute paths,
Windows drive prefixes, invalid UTF-8 text, control characters, `..`, `\`,
`:`, Windows-illegal characters, reserved device basenames, empty segments such
as `a//b.hwp`, `.` segments such as `a/./b.hwp`, and trailing space/dot
segments instead of slash-normalizing malformed payloads.
macOS SMB URLs must escape `smb_share` as its own path segment and escape each
`relative_path` segment separately.
macOS mount discovery and local mount overrides must verify the selected mount
root matches the expected SMB host/share when the OS exposes mount identity. If
the identity is unavailable or mismatched, clients must fail with explicit
mount-unverified or mount-mismatch errors instead of opening from that path.
Desktop clients must call OS open/reveal through argument-safe APIs or argv
lists, never shell strings built from derived paths.

Minimal Samba share shape:

```ini
[documents]
   path = /data/documents
   browseable = yes
   read only = no
   guest ok = no
   valid users = @docsearch-users
```

The exact Samba settings may differ by site, but the share path must expose the
same files that DocSearcher indexes under `document_root.server_path`.

Operator checklist:

- Confirm each `document_root.server_path` exists on the Linux server and is the
  same directory exposed by the named Samba share.
- Confirm each `smb_host` is a host only and each `smb_share` is one share name,
  not a URL, UNC path, host with port, IPv6 literal, or path with extra
  segments.
- Confirm clients can reach `smb_host` on TCP 445 through the private network.
- Confirm a Windows client can open `\\smb_host\smb_share` before testing
  DocSearcher open actions.
- Confirm a macOS client can mount `smb://smb_host/smb_share` before testing
  Wails open actions.
- Keep local drive letters and `/Volumes/...` paths out of server config; those
  belong in client-local mount overrides only.
- Verify client-local mount overrides against the expected SMB host/share when
  the OS exposes mount identity before using them for open/reveal.
- For Windows UNC overrides, require the normalized host/share to match the
  root's `smb_host`/`smb_share` or one normalized `smb_aliases` pair for that
  root.
- For Windows drive-letter overrides, require explicit user/operator approval
  for the root because the drive path cannot prove SMB host/share identity;
  still verify containment before open/reveal.

