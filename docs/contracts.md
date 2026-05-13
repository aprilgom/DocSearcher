# Contracts

This document is the discoverable contract index for DocSearcher HTTP fragments
and persisted configuration. The current UI uses HTMX, so most API responses are
HTML fragments rather than JSON DTOs.

## HTTP Endpoints

| Method | Path | Inputs | Response | Owner |
| --- | --- | --- | --- | --- |
| `GET` | `/` | none | Full HTML page from `web/templates/index.html` | `internal/server.homeHandler` |
| `GET` | `/api/search` | Query params: `q`, `exact=true`, `nospace=true` | Search result fragments plus out-of-band `#search-time` update | `internal/server.searchHandler` |
| `GET` | `/api/config` | none | `<li>` fragments for watched paths | `internal/server.configHandler` |
| `POST` | `/api/watch` | Form field: `path` | Re-rendered watched path list | `internal/server.watchHandler` |
| `DELETE` | `/api/watch` | Query param: `path` | Re-rendered watched path list | `internal/server.watchHandler` |
| `GET` | `/api/stats` | none | `<span>` fragment: document count, watched path count, indexing status | `internal/server.statsHandler` |
| `POST` | `/api/index/reset` | none | Status fragment; starts re-indexing configured watched paths | `internal/server.resetHandler` |

## Search Parameters

`/api/search` accepts these query parameters:

| Name | Type | Default | Meaning |
| --- | --- | --- | --- |
| `q` | string | empty | User search text passed to `internal/infra/search.Search`. |
| `exact` | boolean string | `false` | When `true`, uses phrase matching against `content`. |
| `nospace` | boolean string | `false` | When `true`, searches `content_nospace`; this takes precedence over `exact`. |

## Persisted Config

The committed config contract is `config.example.json`. Local runtime config is
stored in ignored `config.json`.

```json
{
  "watched_paths": []
}
```

| Field | Type | Owner | Notes |
| --- | --- | --- | --- |
| `watched_paths` | `string[]` | `internal/infra/config.Config.WatchedPaths` | Absolute or user-provided directory paths watched recursively by `internal/infra/watcher`. |

## Search Index Document

Documents written to Bleve by `internal/infra/search.IndexDocument` have these fields:

| Field | Type | Source | Notes |
| --- | --- | --- | --- |
| `content` | string | `internal/infra/parser.Parse` | Extracted document text, analyzed with the custom n-gram analyzer. |
| `content_nospace` | string | `domain.NewIndexedDocument` | `content` with spaces, tabs, CR, and LF removed for ignore-space search. |
| `path` | string | indexed file path | Stored path used by the UI to display and open the source file. |
