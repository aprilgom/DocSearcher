# Cross-Platform Name Caveats

The Linux server may allow two different files whose names differ only by case,
while Windows or macOS SMB clients may present those names ambiguously depending
on share and filesystem settings.

- The first implementation does not deduplicate case-only filename collisions.
- Operators should avoid storing case-only sibling document names in shared
  roots that are opened from Windows or macOS clients.
- If a collision is detected during indexing or diagnostics, log it as an
  operational warning instead of silently rewriting logical IDs.
