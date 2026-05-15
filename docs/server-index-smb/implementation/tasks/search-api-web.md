# Search API And Web UI

- [ ] Return search hits with logical path fields.
- [ ] Update HTMX search result fragments to carry `data-document-id`,
  `data-root-id`, `data-root-revision`, and `data-relative-path` for desktop
  bridge actions.
- [ ] HTML-escape displayed paths and `data-*` attributes so filenames with
  quotes, markup characters, Korean text, spaces, `#`, or `%` cannot break the
  fragment or bridge payload.
- [ ] Add a root metadata endpoint or equivalent payload for desktop clients to
  resolve `root_id` into SMB share metadata.
- [ ] Include a root metadata `revision` in `GET /api/document-roots`.
- [ ] Define `revision` as a server-generated opaque value for the whole root
  metadata set; change it whenever any root metadata changes, including
  add/remove, `id`, `name`, `server_path`, `smb_host`, `smb_share`, or
  `smb_aliases`, or validation-affecting canonicalization.
- [ ] Ensure root metadata responses expose only client-safe fields and never
  return `server_path`; if aliases are configured, expose only alias host/share
  pairs.
- [ ] Include the root metadata revision used for rendering in JSON search
  responses and HTMX fragments.
- [ ] Display root name and relative path instead of raw server absolute paths.
- [ ] Keep search responses and open payloads free of `server_path`.
- [ ] Add browser-only copy actions for logical path, SMB URL, or UNC path only
  when they can be derived without exposing server-local paths.
- [ ] Revalidate raw `relative_path` from API responses, HTMX `data-*`
  attributes, cached DOM state, or logical IDs before browser copy derives SMB
  URLs or UNC paths; reject hostile or stale payloads with absolute paths,
  Windows drive prefixes, invalid UTF-8 text, control characters, `..`, `.`,
  `\`, `:`, Windows-illegal characters, reserved device basenames, empty
  segments such as `a//b.hwp`, or trailing space/dot segments. Do not
  slash-normalize malformed client payloads such as `a/./b.hwp`.
- [ ] Derive SMB URLs by URL-escaping `smb_share` as one path segment and each
  `relative_path` segment separately.
- [ ] Keep browser-only result actions separate from desktop-client open/reveal
  actions.
- [ ] Add an explicit desktop bridge capability check for logical open/reveal
  payloads before showing native actions.
- [ ] Update README, `docs/contracts.md`, and `ARCHITECTURE.md` references that
  still describe `watched_paths` as the active configuration contract.

