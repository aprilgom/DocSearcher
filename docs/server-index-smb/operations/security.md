# Security Model

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

