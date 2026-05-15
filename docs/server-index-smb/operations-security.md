# Server Index SMB Operations And Security

## Target Deployment

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

## Deployment Runbook

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

## Operational Diagnostics

Useful checks before blaming DocSearcher open behavior:

```bash
test -d /data/documents
smbclient -L //docserver -U '<user>'
smbclient //docserver/documents -U '<user>' -c 'ls'
```

From Windows:

```powershell
Test-Path '\\docserver\documents'
```

From macOS:

```bash
open 'smb://docserver/documents'
```

These checks validate network reachability and Samba permissions. They do not
replace DocSearcher tests for logical IDs, search hydration, or desktop bridge
payloads.

Permission mismatches are possible and should be diagnosed explicitly:

- If the DocSearcher server process can read a file but the SMB user cannot,
  search may find the document while desktop open fails with a permission error.
- If an SMB user can write a file but the DocSearcher server process cannot read
  it afterward, server-side re-indexing should fail visibly in logs and leave the
  previous searchable content until a successful re-index.
- Operators should align Linux filesystem ownership, Samba user/group mapping,
  and DocSearcher service account permissions for each document root.
- Operators should treat root removal as a maintenance action: rebuild the index
  or run reconciliation before relying on search results, because stale hits for
  removed roots cannot be opened safely.

## Security Model

SMB must not be exposed directly to the public internet. Tailscale should provide
private network reachability, and SMB access should be limited at both the
Tailscale and Samba layers.

Recommended controls:

- Allow SMB only through the private network or Tailscale interface.
- Use Tailscale ACLs or grants to limit access to the document server's SMB port.
- Disable SMBv1.
- Use SMB2/SMB3, with signing and encryption where practical.
- Disable guest and anonymous access.
- Use user or group based Samba permissions.
- Separate read-only users from editors.
- Keep server and clients updated.
- Keep backups or snapshots for document recovery.

Example Tailscale ACL intent:

```json
{
  "action": "accept",
  "src": ["group:docsearch-users"],
  "dst": ["docserver:445"]
}
```

Use the site's current Tailscale policy format and host names; the key point is
that only approved users and clients should reach SMB.

DocSearcher must respect filesystem permissions. If SMB access fails, the client
should surface a mount, permission, or availability error.

## Data Exposure Rules

- Logs may include `server_path` when needed for server-side diagnostics.
- Browser responses, root metadata responses, copy actions, and desktop open
  payloads must not include `server_path`.
- Error messages shown to desktop users may include resolved UNC, SMB URL, or
  local mounted paths for the file being opened.
- Error messages must not include credentials, environment variables, unrelated
  local paths, or Samba administrative configuration.
