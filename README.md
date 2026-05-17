# Local Audiobooks for Continuum

`continuum.local-audiobooks` scans local audiobook folders and exposes them to
the Continuum Audiobooks portal as an `audiobook_backend.v1` source. It is the
right backend when your audiobook files live on disk next to the Continuum
deployment.

The user-facing web app, playback UI, requests table, and ABS-compatible API
come from `continuum.audiobooks`; this plugin owns local library scanning,
metadata, cover data, and byte-range streaming.

## Detailed Operations Docs

- [Setup, debugging, and communication flows](docs/setup-debug-flows.md)

## Features

- Scans configured library paths for audiobook files.
- Exposes catalog, search, detail, cover, and streaming behavior to the
  Audiobooks portal.
- Supports standalone signed streaming for reverse-proxied direct clients.
- Aggregates metadata from Audnexus, AudiMeta, iTunes, Storytel, BookBeat,
  Audioteka, and Audiobookcovers.
- Caches metadata source results and supports scheduled enrichment.
- Supports inline enrichment after scans when enabled.

## Configuration

| Key | Required | Description |
|---|---|---|
| `database_url` | yes | Postgres DSN for the `local_audiobooks` schema. |
| `library_paths` | yes | JSON array of absolute folders to scan, for example `["/srv/audiobooks"]`. |
| `standalone_http_listen` | no | Optional listener for presigned byte-range streaming. |
| `stream_signing_secret` | conditional | 32-byte base64 HMAC secret shared with the Audiobooks portal when standalone streaming is enabled. |
| `metadata_sources_enabled` | no | JSON array of metadata source IDs to query. Defaults to all. |
| `metadata_default_region` | no | Default ISO country code. Defaults to `us`. |
| `metadata_cache_ttl_days` | no | Positive metadata cache TTL. Defaults to 30 days. |
| `metadata_rate_limit_rps` | no | Per-source request rate limit. |
| `scan_inline_enrich` | no | Run enrichment synchronously after each scan. |
| `metadata_scan_source` | no | Source used by the enrichment worker during scans. |

Example DSN:

```text
postgres://plugin_local_audiobooks:password@postgres:5432/continuum?search_path=local_audiobooks&sslmode=disable
```

## Database Setup

```sql
CREATE ROLE plugin_local_audiobooks WITH LOGIN PASSWORD '<chosen>';
CREATE SCHEMA local_audiobooks AUTHORIZATION plugin_local_audiobooks;
GRANT CONNECT ON DATABASE continuum TO plugin_local_audiobooks;
```

## Portal Integration

1. Mount audiobook files into the plugin runtime.
2. Configure `library_paths`.
3. Install and configure `continuum.audiobooks`.
4. Add a presentation library in the Audiobooks admin UI backed by
   `continuum.local-audiobooks`.

## Build And Test

```bash
make build
make test
```
