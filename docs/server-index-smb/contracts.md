# Server Index SMB Contracts

This file is the contract map for the server-index-SMB design. The detailed contracts are split by responsibility so each file stays reviewable.

## Contract Files

1. [Data Model](contracts/data-model.md) - logical document identity, validation, and path containment.
2. [Server Config](contracts/server-config.md) - `document_roots`, SMB metadata, aliases, and root lifecycle rules.
3. [Client Config](contracts/client-config.md) - desktop mount/override configuration and client-side path validation.
4. [API Contract](contracts/api-contract.md) - root metadata, search responses, HTMX attributes, and revision handling.
5. [Open Flow](contracts/open-flow.md) - Windows/macOS open/reveal sequencing and error classification.
6. [Indexing And Watch Rules](contracts/indexing-watch.md) - root ownership, watcher handling, reconciliation, and stale cleanup.
7. [Bleve Storage Contract](contracts/bleve-storage.md) - logical IDs, stored fields, and diagnostic `server_path` restrictions.
8. [Cross-Platform Name Caveats](contracts/name-caveats.md) - case-only filename collision guidance.

## Core Contract

- The server indexes Linux files, but clients open through SMB-derived paths.
- `server_path` is server-only operational state and must not be treated as client-openable.
- Client-visible identity is `root_id`, `root_revision`, and `relative_path` plus client-safe root SMB metadata.
- Raw client payloads are revalidated before copy/open/reveal path derivation.
- Root metadata revisions and SMB alias rules prevent stale or wrong-share opens.
