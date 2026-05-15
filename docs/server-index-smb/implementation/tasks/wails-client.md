# Wails Desktop Client

- [ ] Let Wails coexist with `cmd/client` until Windows behavior is
  feature-complete, then retire `cmd/client` in a separate cleanup.
- [ ] Resolve Windows open paths from `smb_host`, `smb_share`, and
  `relative_path` into UNC paths.
- [ ] Support drive-letter mounts as optional Windows overrides.
- [ ] Require Windows UNC overrides to match the expected normalized
  `smb_host`/`smb_share` or any normalized `smb_aliases` pair configured for
  that root; reject wrong-host or wrong-share overrides.
- [ ] Require explicit user/operator configuration for Windows drive-letter
  overrides because the path alone cannot prove host/share identity; still
  enforce containment after joining.
- [ ] Derive SMB URLs with URL path escaping, and derive UNC paths without URL
  escaping by using the OS-independent Windows/UNC helper.
- [ ] Refresh root metadata before desktop open/reveal when the cached revision
  is missing or differs from the search result/fragment revision.
- [ ] Revalidate raw `relative_path` payloads before desktop open/reveal path
  derivation so malicious or stale client data fails closed before UNC, SMB URL,
  or mount path derivation.
- [ ] Invoke OS open/reveal through argument-safe APIs or argv lists, never by
  concatenating derived paths into shell command strings.
- [ ] Resolve macOS open paths by finding or mounting `smb://host/share`, then
  joining the local mounted path with `relative_path`.
- [ ] Verify macOS-discovered mounts and client-local mount overrides match the
  expected `smb_host` and `smb_share` when the OS exposes mount identity.
- [ ] Fail macOS open/reveal with explicit mount-unverified or mount-mismatch
  errors when mount identity cannot be verified or does not match, instead of
  opening from the wrong share.
- [ ] Reject path joins that escape the configured mount root.
- [ ] Open files through OS-native file association/default app behavior.
- [ ] Add "show in folder" behavior.
- [ ] Add clear open failures for missing root metadata, missing SMB metadata,
  missing local mount overrides, unavailable files, denied permissions, and
  OS open/reveal failures.
- [ ] On share availability, resolution, or missing-file failures, refresh root
  metadata once, retry open/reveal error classification with fresh metadata, and
  then report the final error without further retries.

