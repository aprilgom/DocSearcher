# Client Verification

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

