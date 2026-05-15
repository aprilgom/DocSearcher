# Server Index SMB Implementation Plan

This file is the implementation-plan map for the server-index-SMB design. Detailed planning is split so sequencing, task checklists, and migration guidance can be reviewed independently.

## Plan Files

1. [Overview And Milestones](implementation/overview.md) - expected code impact, Wails direction, first milestone, and milestone scope.
2. [Implementation Tasks](implementation/tasks.md) - planning decisions and detailed task checklist.
3. [Migration And Questions](implementation/migration.md) - migration steps, closed follow-up questions, and remaining open questions.

## Selected Strategy

- Milestone 1 is server/API/logical payload only: `document_roots`, logical IDs, root metadata API, search payloads, HTMX data attributes, and index rebuild.
- Native open/reveal remains gated until the active desktop bridge supports `root_id`, `root_revision`, and `relative_path`.
- Windows UNC open and Wails cross-platform client work follow after the server contract is stable and test-covered.
- Existing absolute-path Bleve IDs are not migrated in place; operators rebuild or reconcile the index.
