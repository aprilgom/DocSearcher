# Parser Boundary

DocSearcher treats parsing as an adapter boundary. The core application needs
extracted text for searchable documents; it should not depend on HWP/HWPX or PDF
format internals.

## Boundary Owner

`internal/parser` owns file-extension dispatch and converts supported files into
plain text.

| Extension | Adapter | Dependency |
| --- | --- | --- |
| `.hwp` | `internal/parser.parseHWP` | `goHwpTxt.ExtractText` |
| `.hwpx` | `internal/parser.parseHWP` | `goHwpTxt.ExtractText` |
| `.pdf` | `internal/parser.parsePDF` | `github.com/ledongthuc/pdf` |

`goHwpTxt` is a local replacement module, but it should be reviewed as an
external parser dependency. Application changes should start in
`internal/parser` unless the requested behavior explicitly targets HWP/HWPX
parser internals.

## Parser-Facing Contract

`parser.Parse(path)` returns:

| Case | Result |
| --- | --- |
| Supported file with readable text | Extracted UTF-8 text |
| Supported file with parser failure | Non-nil error from the adapter |
| Unsupported extension | Non-nil `unsupported file type` error |

The parser layer does not index, normalize spacing, walk watched folders, watch
files, or render UI. Those responsibilities belong to `internal/app`,
`internal/domain`, `internal/scanner`, `internal/watcher`, and
`internal/server`.

## Fixture Expectations

Parser fixtures should be synthetic or explicitly safe to commit. Do not commit
real user documents under `goHwpTxt/testdata/`.

When parser behavior changes, prefer fixture coverage for:

| Behavior | Expected evidence |
| --- | --- |
| Extension dispatch | Tests in `internal/parser` for supported and unsupported extensions |
| HWP/HWPX extraction changes | Focused tests in `goHwpTxt` plus `internal/parser` integration coverage when the app contract changes |
| PDF extraction changes | Tests or sample fixtures that prove text extraction errors are surfaced |
| Empty text handling | Explicit assertion for whether empty text is accepted or rejected by the app layer |

## Change Guidance

1. Keep app-facing parser behavior in `internal/parser`.
2. Change `goHwpTxt` only when the HWP/HWPX format behavior itself must change.
3. After touching `goHwpTxt`, run both root package tests and
   `cd goHwpTxt && go test ./...`.
4. Avoid broad formatting-only edits to table data such as
   `goHwpTxt/pkg/hwp3/hnc2unicode_tables.go`.
