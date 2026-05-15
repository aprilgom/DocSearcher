# API Contract

Root metadata is a first-class API contract, not an implementation detail.
`GET /api/document-roots` returns client-safe root metadata and a revision for
the root metadata set. The endpoint should be available to the browser UI and
desktop clients through the same server origin as search:

```json
{
  "revision": "roots-0007",
  "roots": [
    {
      "id": "documents",
      "name": "문서 공유",
      "smb_host": "docserver",
      "smb_share": "documents",
      "smb_aliases": [
        { "host": "dfs-docs", "share": "documents" }
      ]
    }
  ]
}
```

The root metadata API may expose only client-safe alias host/share pairs in
`smb_aliases`. It must not return `server_path`.

`revision` is server-generated metadata for the whole document-root metadata
set. It is opaque to clients and must change whenever any document root metadata
changes, including adding or removing a root, changing `id`, `name`,
`server_path`, `smb_host`, `smb_share`, or `smb_aliases`, or changing
validation-affecting canonicalization. Clients compare it only for equality as a
string or number and must not infer ordering, timestamps, or semantic meaning
from its value.

Clients may cache root metadata for responsiveness, but must track the
`revision`. Search responses and rendered HTMX fragments carry the root metadata
revision used for rendering. Before open/reveal, clients must refresh root
metadata when their cached revision is missing or differs from the search
result/fragment revision. They must also refresh when a search hit references an
unknown `root_id` or when an open/reveal attempt fails because root metadata is
missing. If open/reveal fails because the share is unavailable, resolution
fails, or the resolved file is missing, refresh root metadata once and retry
classification with fresh metadata before reporting the final error. A stale
cache must not cause fallback to a server-local path.

Search results should return logical file identity:

```json
{
  "id": "documents:shared/2026/sample.hwp",
  "root_id": "documents",
  "root_name": "문서 공유",
  "root_revision": "roots-0007",
  "relative_path": "shared/2026/sample.hwp",
  "fragment": "..."
}
```

Expected behavior:

- The browser UI displays the root name and relative path.
- The desktop client receives `root_id`, `root_revision`, and `relative_path`
  for open/reveal actions.
- The API must not treat Linux `server_path` values as client-openable paths.
- The desktop client reads `smb_host` and `smb_share` from the root metadata API
  by `root_id`.
- Search hits may include display metadata such as `root_name`, but they must
  remain logical and compact.
- Search hits and open payloads must not include `server_path`.
- Search hit `id` is the logical `document_id`. Clients should pass it around as
  an opaque identifier and use `root_id` plus `relative_path` for path
  resolution.
- Existing `/api/watch` UI/API behavior should be replaced by document-root
  management that edits or reports `document_roots`. New UI copy should use
  "document root" semantics rather than local watch-path semantics.
- Browser-only usage may show result metadata and copyable paths, but native
  open/reveal actions are available only through a desktop client bridge.
- Browser-only UI may show copy actions for `relative_path`, SMB URL, or UNC
  path when root metadata allows deriving them. It must derive SMB URLs with URL
  path escaping for the share segment and each relative-path segment, and derive
  UNC paths with the OS-independent Windows/UNC helper rather than string
  concatenation or the build host's path joining. It should not show native
  open/reveal buttons unless a desktop bridge is connected.
- When the current HTMX search flow renders HTML fragments, each result element
  should carry `data-document-id`, `data-root-id`, `data-root-revision`, and
  `data-relative-path` attributes for the desktop bridge. The fragment must not
  embed `server_path`.
- HTMX fragments must HTML-escape displayed paths and attribute values. A
  filename containing spaces, Korean characters, quotes, `<`, `>`, `&`, `#`, or
  `%` must not break markup, attributes, copy actions, or desktop bridge
  payloads.
- If a JSON search API is added for the Wails client, it should use the same
  logical fields as the example above and avoid a separate path contract.
- Desktop bridge calls should include an explicit capability version or feature
  flag for logical open payloads. The browser UI should enable native
  open/reveal only after that capability is observed.

