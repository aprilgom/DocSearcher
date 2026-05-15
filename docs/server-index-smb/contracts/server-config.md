# Server Config

Server-side runtime config describes indexed document roots:

```json
{
  "document_roots": [
    {
      "id": "documents",
      "name": "문서 공유",
      "server_path": "/data/documents",
      "smb_host": "docserver",
      "smb_share": "documents",
      "smb_aliases": [
        { "host": "dfs-docs", "share": "documents" }
      ]
    }
  ]
}
```

`document_roots` is shared operational state. It defines what the server scans
and indexes.

Minimal valid examples:

```text
id:          documents
name:        문서 공유
server_path: /data/documents
smb_host:    docserver
smb_share:   documents
```

Invalid examples:

```text
id:          Documents        # uppercase
id:          docs/main        # slash
relative:    ../secret.hwp    # parent escape
relative:    C:/docs/a.hwp    # Windows drive prefix
relative:    a:b.hwp          # ambiguous document_id separator
relative:    a\b.hwp          # backslash changes UNC path semantics
relative:    folder./a.hwp     # Windows file APIs/UNC may trim or reject segment
smb_host:    user@docserver   # credentials/authority syntax
smb_host:    docserver:445     # ports are not supported in Milestone 1
smb_host:    [fd7a::1]         # IPv6 literals are not supported in Milestone 1
smb_host:    docserver?x      # URL query delimiter
smb_host:    docs%31          # percent-encoded host syntax is unsupported
smb_host:    문서서버          # non-ASCII/IDNA is unsupported in Milestone 1
smb_share:   docs/shared      # share name, not a path
smb_share:   docs#archive     # URL fragment delimiter
smb_share:   docs:archive     # UNC/path metacharacter
smb_share:   docs*archive     # UNC/path metacharacter
```

Server config rules:

- `id` must satisfy the `root_id` validation rules and be unique.
- `server_path` must be absolute after `filepath.Clean`.
- `name` is display metadata. Changing it must not require rebuilding the index.
- Changing a root `id` creates a new logical namespace and requires rebuilding
  the affected index documents.
- Changing `server_path` changes the indexed source of truth and requires
  re-indexing that root. If the old and new paths both exist, remove stale
  documents from the old root namespace during the same maintenance window.
- `smb_host` is the ASCII hostname, DNS name, Tailscale name, or IP address
  clients use to reach the Samba server.
- `smb_share` is the Samba share name, which may differ from the final
  `server_path` folder name.
- `smb_host` and `smb_share` are trimmed before validation and must remain
  non-empty after trimming.
- `smb_host` must be a host value only. It must not contain whitespace, control
  characters, non-ASCII text, `%`, `:`, `/`, `\`, `@`, `?`, or `#`, and must
  not include URL credentials or authority syntax. Milestone 1 supports only
  ASCII hostnames, IP addresses, and Tailscale names. IDNA, ports, and IPv6
  literals are unsupported until a later implementation explicitly normalizes
  them.
- `smb_share` must be a single share name, not a path. It must not contain
  whitespace, control characters, `/`, `\`, `?`, `#`, or UNC/path
  metacharacters (`:`, `*`, `"`, `<`, `>`, `|`).
- `smb_aliases` is optional per-root server metadata shaped as an array of
  `{ "host": "...", "share": "..." }` pairs. Aliases are operator-owned config
  because they authorize alternate shares, DFS names, or hostnames for the same
  root. Alias `host` and `share` values are trimmed, validated, and normalized
  with the same rules as `smb_host` and `smb_share`.
- Windows UNC paths are derived from `smb_host`, `smb_share`, and
  `relative_path` with an explicit Windows/UNC-semantics helper that is
  OS-independent and testable on macOS/Linux.
- macOS SMB URLs are derived from `smb_host`, `smb_share`, and `relative_path`;
  the desktop client still opens files through the mounted filesystem path.
- Overlapping roots are allowed. If multiple roots contain the same file, the
  most specific matching root owns that file.
- Two configured roots must not resolve to the exact same canonical
  `server_path`. Reject duplicate canonical roots during config validation.
- A parent root scan must skip subtrees owned by more specific child roots so
  the same physical file is not indexed under multiple logical IDs.
- `document_roots` is the only supported root configuration for this design.
  Existing `watched_paths` config is not migrated in place.
- `watched_paths` may remain in older local config files, but new SMB-index code
  must not silently convert it into document roots because root IDs and SMB share
  metadata are operator decisions.
- The committed `config.example.json` should move to this `document_roots`
  shape in the same implementation slice that introduces server-side
  `document_roots` loading.
- DocSearcher does not parse Samba configuration files. Operators must ensure
  that `smb_host` and `smb_share` expose the configured `server_path`. Optional
  diagnostics may test access to derived SMB paths, but config validation should
  not require Samba administrator access.
- Removing a root from config makes its indexed documents stale. The next
  maintenance window must remove documents for that `root_id` or rebuild the
  index, otherwise old search hits may point to roots no longer exposed by the
  root metadata API.

