# Server Index SMB Verification

## Server Verification

- Unit test logical ID creation from `root_id`, `server_path`, and file path.
- Unit test `root_id` rejects `:`, slashes, whitespace, empty values, and
  uppercase or non-ASCII values.
- Unit test `relative_path` rejects absolute paths, Windows drive prefixes,
  `..` segments, `.` segments such as `a/./b.hwp`, `:`, `\`,
  Windows-illegal filename characters (`*`, `?`, `"`, `<`, `>`, `|`), and
  empty path segments.
- Unit test `relative_path` rejects invalid UTF-8 text and control characters.
- Unit test `relative_path` rejects reserved Windows device basenames (`CON`,
  `PRN`, `AUX`, `NUL`, `COM1`-`COM9`, `LPT1`-`LPT9`) case-insensitively,
  including names with extensions such as `CON.txt`.
- Unit test `relative_path` rejects any path segment ending in a space or `.`
  because Windows file APIs and UNC opens may trim or reject those names.
- Unit test indexing skips a supported file whose computed `relative_path` is
  not valid UTF-8 text or is not SMB-open-safe and logs an operational warning
  without encoding, replacing, or rewriting the path.
- Unit test slash-normalized `relative_path` storage.
- Unit test logical IDs preserve UTF-8 `relative_path` text without URL
  encoding, including spaces, Korean characters, `#`, and `%`.
- Unit test logical ID parsing splits on the first `:` and revalidates both
  fields.
- Unit test changing a root `id` is treated as a logical namespace change that
  requires rebuild or reconciliation.
- Integration test changing a root's `server_path` while keeping the same
  `root_id` removes stale indexed documents from the old path during
  reconciliation, or requires a full rebuild before search is trusted.
- Unit test removing a root from config removes or marks stale indexed documents
  for that `root_id` during reconciliation.
- Unit test document root config validates required `smb_host` and `smb_share`
  metadata for SMB-open-enabled roots.
- Unit test document root config rejects duplicate `document_roots[].id`
  values.
- Unit test document root config rejects `server_path` values that are relative
  or non-absolute after `filepath.Clean`.
- Unit test document root config rejects malformed `smb_host` values including
  empty-after-trim, slash, backslash, whitespace, control characters, `:`, `@`,
  `?`, `#`, `%`, and non-ASCII text.
- Unit test document root config rejects `smb_host` values with ports or IPv6
  literal syntax because Milestone 1 does not normalize colon-containing hosts.
- Unit test document root config rejects IDNA hostnames until an explicit
  normalization policy is implemented.
- Unit test document root config rejects malformed `smb_share` values including
  empty-after-trim, slash, backslash, whitespace, control characters, `?`, `#`,
  `:`, `*`, `"`, `<`, `>`, and `|`.
- Unit test optional `smb_aliases` validates each alias host/share pair with
  the same trimming, validation, and normalization rules as `smb_host` and
  `smb_share`.
- Unit test document root config rejects two roots whose `server_path` values
  resolve to the same canonical path.
- Unit test document root config still allows overlapping parent/child roots
  with different canonical paths.
- Unit test search result hydration includes `root_id` and `relative_path`.
- Unit test search result hydration uses stored `root_id` and `relative_path`
  fields rather than parsing the Bleve hit ID as the source of truth.
- Unit test search result rendering attaches current `root_name` from root
  metadata without depending on a stored Bleve `root_name` field.
- Unit test corrupt search hits with missing or invalid stored `root_id` or
  `relative_path` are omitted from user-visible results.
- Unit test search responses and open payloads never include `server_path`.
- Unit test Bleve documents use logical `document_id` as the document ID and
  store `root_id` and `relative_path`.
- Unit test optional Bleve `server_path` is stored-only diagnostics: it is not
  indexed, not queryable, not highlighted, and not used in search response
  hydration.
- Unit test overlapping roots choose the most specific matching root.
- Unit test parent-root scans skip subtrees owned by more specific child roots.
- Integration test overlapping roots do not produce duplicate logical documents
  during a full scan.
- Integration test removing a child root and re-indexing moves affected files to
  the parent root when a parent root still contains them.
- Unit test scan skips symlink file entries without following them.
- Unit test scan skips symlink directory entries without descending into them.
- Unit test config validation resolves a symlink `server_path` root to its
  canonical path before containment and relative-path computation.
- Unit test config validation rejects an unresolved symlink `server_path` root.
- Unit test watcher handling skips symlink entries without indexing their
  targets.
- Unit test an in-root symlink to another in-root document does not create a
  duplicate logical document.
- Unit test Unix containment rejects only `filepath.Rel` results equal to `..`
  or prefixed by `../` after path cleaning, while accepting contained names such
  as `..draft/file.hwp`.
- Unit test Windows containment rejects prefix traps such as `Z:\docs2` under
  `Z:\docs`.
- Unit test Windows containment rejects `\\server\share2\...` under
  `\\server\share`.
- Unit test Windows containment accepts case-only path differences under the
  same drive or UNC share.
- Unit test Windows containment requires matching drive roots or matching UNC
  server/share roots before relative path comparison.
- Unit test old `watched_paths`-only config is rejected or reported as requiring
  explicit `document_roots`.
- Unit test config loading ignores `watched_paths` when `document_roots` is
  present.
- Unit test `config.example.json` matches the committed `document_roots` config
  shape after the new config contract is implemented.
- Unit test stats/reporting uses document-root count semantics instead of
  watched-path count semantics after the server migration.
- Integration test re-index after file change under a configured root.
- Integration test delete events remove the logical document ID without needing
  the deleted file to exist.
- Integration test deleting a directory removes indexed documents under that
  `root_id` and `relative_path` prefix from search results.
- Integration test directory-prefix deletion matches slash components only, so
  deleting `a/b` does not remove indexed documents under `a/b2`.
- Integration test renaming a directory removes old-prefix documents from search
  results and indexes the new-prefix documents when still under a configured
  root.
- Integration test root reconciliation removes indexed documents whose
  `relative_path` no longer exists under the root.
- Integration test root reconciliation removes indexed documents for root IDs no
  longer present in `document_roots`.
- API test `GET /api/document-roots` returns `revision` plus roots containing
  `id`, `name`, `smb_host`, `smb_share`, and optional client-safe
  `smb_aliases`.
- API/unit test root metadata `revision` changes when the root metadata set
  changes, including add/remove, `id`, `name`, `server_path`, `smb_host`,
  `smb_share`, `smb_aliases`, or validation-affecting canonicalization.
- API/unit test clients treat root metadata `revision` as an opaque string or
  number and compare it only for equality, without ordering assumptions.
- API test root metadata responses do not include `server_path` and expose only
  alias host/share pairs for `smb_aliases`.
- API test search responses include the root metadata revision used for
  rendering.
- API test legacy `/api/watch` behavior is replaced or explicitly redirected to
  document-root semantics.
- API or fragment test HTMX search results include `data-document-id`,
  `data-root-id`, `data-root-revision`, and `data-relative-path` attributes
  without embedding `server_path`.
- Fragment test result display text and `data-*` attributes are HTML-escaped for
  filenames containing quotes, `<`, `>`, `&`, spaces, Korean text, `#`, and `%`.
- Documentation check: README, `docs/contracts.md`, `ARCHITECTURE.md`, and
  `config.example.json` no longer describe `watched_paths` as the active config
  contract once the SMB design is implemented.

## Client Verification

- Unit test Windows mount joining from `root_id` and `relative_path`.
- Unit test Windows UNC path derivation from `smb_host`, `smb_share`, and
  `relative_path` uses an explicit OS-independent Windows/UNC-semantics helper
  that is testable on macOS/Linux.
- Unit test Windows UNC path derivation rejects malformed host/share metadata
  before constructing a UNC path.
- Unit test Windows UNC mount joining from `root_id` and `relative_path`.
- Unit test Windows UNC containment rejects `\\server\share2\...` as outside
  `\\server\share`.
- Unit test Windows UNC override rejects a normalized host or share that differs
  from the root metadata.
- Unit test Windows UNC override accepts an operator-approved host/share alias
  from `smb_aliases` for that root after normalization.
- Unit test Windows UNC override rejects wrong-host or wrong-share values that
  match neither the primary root metadata nor any normalized alias pair.
- Unit test Windows drive-letter override requires explicit user/operator
  configuration because identity cannot be proven from `Z:\` alone, and still
  rejects joined paths that escape the configured root.
- Unit test Windows mount containment rejects `Z:\docs2` as outside `Z:\docs`.
- Unit test Windows mount containment allows case-only differences under the
  same root.
- Unit test macOS mount joining from `root_id` and `relative_path`.
- Unit test macOS SMB URL derivation URL-escapes `smb_share` as its own path
  segment, including share names with `%` or non-ASCII characters.
- Unit test macOS SMB URL derivation rejects malformed host/share metadata
  before constructing an SMB URL.
- Unit test path joining rejects cleaned paths that escape the configured mount
  root.
- Unit test desktop open/reveal actions load `smb_host` and `smb_share` from the
  root metadata API by `root_id`.
- Unit test desktop open/reveal refreshes root metadata before path derivation
  when the cached revision is missing or differs from the search result/fragment
  revision.
- Unit test desktop open/reveal refreshes root metadata once for an unknown
  `root_id`, then reports the root as unavailable.
- Unit test desktop open/reveal refreshes root metadata once when cached SMB
  metadata produces a share-unavailable, resolution, or missing-file failure,
  then retries classification with fresh metadata.
- Unit test desktop open/reveal does not refresh root metadata in an infinite
  loop after the retry still fails.
- Unit test browser-only UI hides native open/reveal actions when no desktop
  bridge is connected.
- Unit test browser UI hides native open/reveal actions when the connected
  bridge lacks the logical payload capability.
- Unit test browser-only UI can copy logical path or derived SMB/UNC path
  without exposing server-local paths.
- Unit test browser copy code revalidates raw API/HTMX/DOM/logical-ID
  `relative_path` payloads and rejects absolute paths, Windows drive prefixes,
  invalid UTF-8 text, control characters, `..`, `.`, `\`, `:`,
  Windows-illegal characters, reserved device basenames, empty segments, and
  trailing space/dot segments, including stale payloads such as `a//b.hwp` and
  `a/./b.hwp`, before deriving SMB URLs or UNC paths.
- Unit test SMB URL copy escapes path segments containing spaces, Korean
  characters, `#`, or `%`.
- Unit test SMB URL copy escapes the share segment separately from
  `relative_path` segments, including `smb_share` values containing `%` or
  non-ASCII characters.
- Unit test SMB URL copy also escapes URL path-segment-unsafe characters, such
  as spaces, `#`, `%`, and non-ASCII text, without changing the stored
  `relative_path` contract.
- Unit test UNC copy uses the OS-independent Windows/UNC-semantics helper and
  preserves literal filename characters without URL escaping.
- Unit test desktop bridge open/reveal revalidates raw API/HTMX/DOM/logical-ID
  `relative_path` payloads and rejects absolute paths, Windows drive prefixes,
  invalid UTF-8 text, control characters, `..`, `.`, `\`, `:`,
  Windows-illegal characters, reserved device basenames, empty segments, and
  trailing space/dot segments, including stale payloads such as `a//b.hwp` and
  `a/./b.hwp`, before deriving UNC, SMB URL, or mounted paths.
- Unit test macOS mount discovery accepts a mounted root only when the OS-exposed
  mount identity matches the expected `smb_host` and `smb_share`.
- Unit test macOS client-local mount overrides accept a mounted root only when
  the OS-exposed mount identity matches the expected `smb_host` and `smb_share`.
- Unit test macOS open/reveal fails with mount-mismatch when the selected mount
  root resolves to a different SMB host or share.
- Unit test macOS open/reveal fails with mount-unverified when mount identity is
  unavailable where the OS normally exposes it, instead of opening from an
  unverifiable local path.
- Unit test desktop open/reveal invokes OS operations with argument-safe APIs or
  argv lists, not shell strings, for paths containing spaces, commas, leading
  dashes, quote-like display characters, `#`, `%`, and Korean text.
- Unit test reveal selection for those characters opens the containing folder
  and selects the intended file without command/argument injection or sibling
  mis-selection.
- Unit test missing mount, missing file, and permission/open error messages.
- Unit test open actions are hidden or disabled when the desktop bridge cannot
  consume the new logical open payload.
- Unit test legacy `openFile(path)` behavior is not fed Linux `server_path`
  values after the SMB contract is enabled.
- Manual Windows test: search, double-click, edit in native app, save, close, and
  confirm server-side re-index sees the update.
- Manual Windows UNC test: configure `\\docserver\documents`, open a search hit,
  reveal it in Explorer, edit, save, and confirm the server re-indexes it.
- Manual macOS test after Wails support exists: configure `/Volumes/documents`,
  open a search hit, reveal it in Finder, edit, save, and confirm the server
  re-indexes it.

## Repository Verification

Documentation-only edits:

- Review rendered Markdown and `git diff`.
- Go tests may be skipped when no Go code or config behavior changed. Report
  that no Go code changed.

On macOS/Linux:

```bash
go test $(go list ./... | grep -v '/cmd/client$')
go build ./cmd/app
```

On Windows, also run:

```bash
go run ./cmd/client
```

If `goHwpTxt` is touched, also run:

```bash
cd goHwpTxt && go test ./...
```
