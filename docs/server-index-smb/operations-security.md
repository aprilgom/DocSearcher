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
  Mounts the same share as \\docserver\documents or Z:\
  Opens search hits through the OS shell/default app

macOS client
  Mounts the same share as smb://docserver/documents
  Opens search hits through /Volumes/documents or another mounted path

Tailscale
  Provides private network reachability to the Linux server and SMB port
```

Clients do not build local indexes. The Linux server owns document storage,
filesystem watching, parsing, and indexing.

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

DocSearcher must respect filesystem permissions. If SMB access fails, the client
should surface a mount, permission, or availability error.
