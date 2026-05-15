# Operational Diagnostics

Useful checks before blaming DocSearcher open behavior:

```bash
test -d /data/documents
smbclient -L //docserver -U '<user>'
smbclient //docserver/documents -U '<user>' -c 'ls'
```

From Windows:

```powershell
Test-Path '\\docserver\documents'
```

From macOS:

```bash
open 'smb://docserver/documents'
```

These checks validate network reachability and Samba permissions. They do not
replace DocSearcher tests for logical IDs, search hydration, or desktop bridge
payloads.

Permission mismatches are possible and should be diagnosed explicitly:

- If the DocSearcher server process can read a file but the SMB user cannot,
  search may find the document while desktop open fails with a permission error.
- If an SMB user can write a file but the DocSearcher server process cannot read
  it afterward, server-side re-indexing should fail visibly in logs and leave the
  previous searchable content until a successful re-index.
- Operators should align Linux filesystem ownership, Samba user/group mapping,
  and DocSearcher service account permissions for each document root.
- Operators should treat root removal as a maintenance action: rebuild the index
  or run reconciliation before relying on search results, because stale hits for
  removed roots cannot be opened safely.

