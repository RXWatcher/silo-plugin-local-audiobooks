# continuum-plugin-audiobooksdb

Local-filesystem audiobook backend for Continuum. Scans a directory tree
of `.m4b` and `.mp3` files and exposes them to the audiobooks portal via
the `audiobook_backend.v1` advertised capability. No ffmpeg dependency at
runtime; M4B + MP3 parsing is pure Go.

## What it does

- Walks one or more configured library paths looking for `.m4b` / `.mp3`
  files.
- Extracts title, author, narrator, year, genre, chapters, cover art, and
  duration from each file. Tags are never modified on disk.
- Serves catalog browsing, cover art (with thumb/medium resize), and
  M4B/MP3 byte-range streaming to the audiobooks portal.
- Optionally serves byte-range streams directly to mobile clients via a
  presigned-URL standalone listener (see `docs/operations.md`).

## Metadata provider (v0.2.0+)

The plugin also advertises the `metadata_provider.v1` capability and acts as
a metadata aggregator for audiobooks. Seven sources are bundled:

| Source ID         | Service           |
|-------------------|-------------------|
| `audnexus`        | Audnexus          |
| `audimeta`        | AudiMeta          |
| `itunes`          | iTunes            |
| `storytel`        | Storytel          |
| `bookbeat`        | BookBeat          |
| `audioteka`       | Audioteka         |
| `audiobookcovers` | Audiobookcovers   |

**Trigger model**: gRPC `Search` queries all enabled sources in parallel and
aggregates results by confidence score. Scan-time enrichment goes through a
Postgres-backed queue drained every minute by the `metadata_enrichment_worker`
scheduled task, using a single configurable source per audiobook (default:
`audnexus`).

**New config keys** (all optional):

| Key | Default | Description |
|-----|---------|-------------|
| `metadata_sources_enabled` | all sources | JSON array of source IDs to query |
| `metadata_default_region` | `"us"` | ISO country code for source requests |
| `metadata_cache_ttl_days` | `30` | Days a positive cache entry is retained |
| `metadata_rate_limit_rps` | `5` | Per-source request rate limit |
| `scan_inline_enrich` | `false` | Run enrichment inline after scan completes |
| `metadata_scan_source` | `"audnexus"` | Source used by the enrichment worker |

**Admin endpoint**: `POST /admin/metadata/backfill` enqueues all unenriched
audiobooks for re-enrichment.

## What it does NOT do

- It does not modify your files. No tag rewrites, no moves, no renames.
- It does not accept book requests. A local backend cannot acquire new
  content — the audiobooks portal handles request routing through other
  backends.

## Format coverage

| Format | Tags | Chapters | Cover | Duration |
|--------|------|----------|-------|----------|
| M4B    | ID3-style atoms via `dhowden/tag` | `chap` atom or synthesized | covr atom or sidecar | mvhd timescale |
| MP3    | ID3v2 frames via `dhowden/tag`   | synthesized (CHAP frame parsing is a follow-up) | APIC frame or sidecar | ID3v2 `TLEN` frame when present |

## Quick start

1. Provision Postgres role + schema (see `docs/operations.md`).
2. Configure `database_url` and `library_paths` in the plugin admin UI.
3. Click "Scan now" in the audiobooks portal admin page.

For full operator setup including the optional presigned-URL streaming
hostname, see [`docs/operations.md`](docs/operations.md).
