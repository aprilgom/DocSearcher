# Open Flow

```text
1. User searches from desktop client.
2. Server returns root_id=documents and relative_path=shared/2026/sample.hwp.
3. Client loads root metadata for documents and refreshes it first if the
   cached revision is missing or differs from the search hit revision.
4. Client revalidates the raw `relative_path` payload and rejects hostile,
   malformed, or stale values, including `.` segments such as `a/./b.hwp`.
5. For macOS mount discovery or local mount overrides, the client verifies the
   selected mount root's exposed SMB identity against the expected
   `smb_host`/`smb_share` when the OS provides that identity, and fails with
   mount-unverified or mount-mismatch if it cannot prove the match.
6. For Windows overrides, UNC roots must match the expected normalized
   `smb_host`/`smb_share` or a normalized `smb_aliases` pair, while
   drive-letter roots require explicit root-level configuration and containment
   only.
7. Windows clients derive a UNC path from fresh `smb_host`, `smb_share`, and
   validated `relative_path`; macOS clients find or create a local mount from
   the SMB URL, then invoke OS-native file association/reveal APIs for the
   resulting path.
```

Windows:

```text
documents -> \\docserver\documents
shared/2026/sample.hwp -> \\docserver\documents\shared\2026\sample.hwp
```

macOS:

```text
smb://docserver/documents -> /Volumes/documents
shared/2026/sample.hwp -> /Volumes/documents/shared/2026/sample.hwp
```

Error handling:

- Missing root metadata: tell the user the server did not return metadata for
  the search hit's `root_id`.
- Stale root metadata: refresh root metadata once, then report that the root is
  no longer available if the `root_id` is still missing.
- Missing SMB metadata: tell the user the root has no usable SMB host/share
  metadata.
- Missing local mount override: tell macOS users, or Windows users who selected
  a local override, that the client has no usable mount for that root.
- Missing file: tell the user the resolved SMB path does not exist or is not
  currently available.
- Permission failure: report that the OS denied access to the resolved file.
- Open failure: include enough resolved-path context for support without exposing
  unrelated local machine data.

The client should classify open errors before falling back to a generic failure.
For share availability, resolution, and missing-file failures, it should refresh
root metadata once, retry classification, and then stop; this rule must not loop
indefinitely.

1. Missing root metadata for `root_id`.
2. Stale root metadata after one refresh attempt.
3. Missing SMB share metadata or local mount mapping for the current platform.
4. Resolved path escapes the UNC/share/mount root after cleaning.
5. Resolved SMB path does not exist or the share is unavailable.
6. OS permission denial.
7. OS-native file association or reveal API failure.

Desktop clients may include the resolved UNC path, SMB URL, or local mounted
path in support-oriented errors. They must not include unrelated local paths,
server-only paths, credentials, or environment details.

