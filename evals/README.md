# Codex Evals

This directory tracks lightweight outcome evidence for Codex work on DocSearcher.
It is intentionally docs/stdlib only.

## Current Baseline

- 2026-05-13: 17/100, Codex-Hostile, category G 1/5.
- Scope note: `goHwpTxt` is excluded because it is a local replace of an external
  HWP/HWPX parser package.

```bash
cd evals
python3 -m json.tool codex-readiness-baseline.json
```

## Representative Task Set

Record Codex outcomes against a small set of representative tasks. Keep the task
set stable unless the repository shape changes.

| Task ID | Representative Work | Expected Evidence |
| --- | --- | --- |
| `docs-context-update` | Update Codex-facing docs such as `AGENTS.md` or module context files. | Diff is scoped, links/commands are accurate, docs-only verification is recorded. |
| `parser-change` | Change HWP/HWPX/PDF text extraction behavior. | Parser tests or a documented platform limitation are recorded. |
| `indexer-change` | Change file walking or indexing flow. | Relevant Go tests and expected indexing behavior are recorded. |
| `search-change` | Change Bleve search behavior or query handling. | Search tests or manual query verification are recorded. |
| `server-ui-change` | Change server routes, templates, or web UI behavior. | Server build/test evidence and browser/manual verification are recorded. |

## Outcome Record Structure

Append completed work to a JSON file using this shape. A future outcomes ledger
can be grouped by date, but keep the fields stable.

```json
{
  "task_id": "docs-context-update",
  "date": "2026-05-13",
  "branch": "docs/codex-readiness-improvements",
  "commit": "optional-short-sha-after-commit",
  "result": "pass",
  "verification": [
    "python3 -m json.tool <baseline-json>",
    "wc -l evals/README.md <baseline-json>"
  ],
  "notes": "Scoped docs-only change; Go tests not run because no Go code changed."
}
```

Allowed `result` values:

- `pass`: task completed and verification matched the expected evidence.
- `partial`: task completed with a documented limitation or skipped check.
- `fail`: task did not meet the expected evidence and needs follow-up.

## Pass-Rate Calculation

Use a simple pass rate for the latest stable task set:

```text
pass_rate = pass_count / total_recorded_tasks
```

Count `partial` separately rather than as a pass. When reporting Codex outcome
evidence in a PR, include:

- Number of recorded tasks.
- Pass, partial, and fail counts.
- Pass rate as a percentage.
- Any repeated failure pattern or missing verification gate.

## Recording Rules

- Keep records factual and short.
- Do not include real documents, local runtime data, secrets, or contents from
  `hwp-index.bleve/`.
- If a verification command cannot run, record the exact reason and residual risk.
- For documentation-only changes, state that Go tests were skipped because no Go
  code changed.
- Update the baseline after running the readiness scorer again, not by guessing.
