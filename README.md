# continuum-plugin-local-audiobooks

Local-filesystem audiobook backend for Continuum. Scans a directory tree of `.m4b` and `.mp3` files and exposes them to the audiobooks portal via the `audiobook_backend.v1` advertised capability. Pure-Go format parsing — no ffmpeg dependency at runtime.

From v0.2.0, the plugin also advertises the `metadata_provider.v1` capability and acts as a metadata aggregator over seven external sources.

## What it does

- Walks one or more configured library paths looking for `.m4b` / `.mp3` files.
- Extracts title, author, narrator, year, genre, chapters, cover art, and duration from each file. Tags are **never modified on disk**.
- Serves catalog browsing, cover art (with thumb/medium resize), and M4B/MP3 byte-range streaming to the audiobooks portal.
- Optionally serves byte-range streams directly to mobile clients via a presigned-URL standalone listener.

## What it does NOT do

- It does not modify your files. No tag rewrites, no moves, no renames.
- It does not accept book requests. A local backend cannot acquire new content — the audiobooks portal handles request routing through other backends.

## Capabilities

| Capability | Notes |
|---|---|
| `audiobook_backend.v1` (`local_audiobooks`) | Catalog, search, browse, cover art, byte-range streaming. |
| `http_routes.v1` (`api`) | `/api/v1/*` for portal calls; `/admin/*` for the metadata backfill endpoint. |
| `metadata_provider.v1` (`local_audiobooks_meta`) | Aggregator over Audnexus, AudiMeta, iTunes, Storytel, BookBeat, Audioteka, Audiobookcovers. |
| `scheduled_task.v1` (`library_scan`) | Cron `0 */6 * * *`. |
| `scheduled_task.v1` (`metadata_enrichment_worker`) | Cron `* * * * *`. Drains the enrichment queue. |

## Format coverage

| Format | Tags | Chapters | Cover | Duration |
|--------|------|----------|-------|----------|
| M4B | ID3-style atoms via `dhowden/tag` | `chap` atom or synthesized | `covr` atom or sidecar | `mvhd` timescale |
| MP3 | ID3v2 frames via `dhowden/tag` | synthesized (CHAP frame parsing is a follow-up) | APIC frame or sidecar | ID3v2 `TLEN` frame when present |

## Metadata provider (v0.2.0+)

| Source ID | Service |
|---|---|
| `audnexus` | Audnexus |
| `audimeta` | AudiMeta |
| `itunes` | iTunes |
| `storytel` | Storytel |
| `bookbeat` | BookBeat |
| `audioteka` | Audioteka |
| `audiobookcovers` | Audiobookcovers |

**Trigger model**: gRPC `Search` queries all enabled sources in parallel and aggregates results by confidence score. Scan-time enrichment goes through a Postgres-backed queue drained every minute by the `metadata_enrichment_worker` scheduled task, using a single configurable source per audiobook (default `audnexus`).

**Admin endpoint**: `POST /admin/metadata/backfill` enqueues all unenriched audiobooks for re-enrichment.

## Configuration

| Key | Required | Description |
|---|---|---|
| `database_url` | yes | DSN for the dedicated `local_audiobooks` schema. |
| `library_paths` | yes | JSON array of absolute filesystem paths to scan, e.g. `["/srv/audiobooks"]`. |
| `standalone_http_listen` | no | Bind a TCP listener (e.g. `:7879`) for presigned-URL byte-range streaming. Requires `stream_signing_secret`. |
| `stream_signing_secret` | conditional | 32-byte base64 HMAC. Must match the portal's `cdn_signing_secret`. Required when `standalone_http_listen` is set. |
| `metadata_sources_enabled` | no | JSON array of source IDs (default: all 7). |
| `metadata_default_region` | no | ISO country code (default `us`). |
| `metadata_cache_ttl_days` | no | Positive cache retention (default 30). |
| `metadata_rate_limit_rps` | no | Per-source RPS (default 5). |
| `scan_inline_enrich` | no | Run enrichment synchronously after a scan (default false). |
| `metadata_scan_source` | no | Source used by the enrichment worker (default `audnexus`). |

## Dependencies

- Postgres role + `local_audiobooks` schema.
- Mounted filesystem with `.m4b` / `.mp3` files.
- Outbound HTTP access to enabled metadata sources.

## Install

```sql
CREATE ROLE plugin_local_audiobooks LOGIN PASSWORD '<chosen>';
CREATE SCHEMA local_audiobooks AUTHORIZATION plugin_local_audiobooks;
```

Configure `database_url` and `library_paths` in the plugin admin UI, then click "Scan now" in the audiobooks portal admin page. For full operator setup including the optional presigned-URL streaming hostname, see [`docs/operations.md`](docs/operations.md).

## Build & test

```bash
make build
make test
```

## Status

v0.2.0, beta. Metadata aggregator is new in 0.2.
