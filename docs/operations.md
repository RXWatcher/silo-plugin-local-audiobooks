# Operations — continuum-plugin-local-audiobooks

## 1. Postgres schema setup

The plugin keeps its tables in a dedicated Postgres schema (namespace)
within a shared Continuum database. Create the role and schema before
installing the plugin:

```sql
CREATE ROLE plugin_local_audiobooks LOGIN PASSWORD '<set-something-strong>';
CREATE SCHEMA local_audiobooks AUTHORIZATION plugin_local_audiobooks;
```

The plugin's DSN must set `search_path=local_audiobooks` so its migrations
target that schema. Example DSN:

```
postgres://plugin_local_audiobooks:<pwd>@db.internal:5432/continuum?search_path=local_audiobooks&sslmode=disable
```

The migrations are idempotent — running them again against an already-
migrated schema is a no-op (tracked via `schema_migration` in that same
schema).

## 2. Library paths

Configure one or more absolute paths in the plugin admin UI as a JSON
array:

```json
["/srv/audiobooks", "/mnt/extra"]
```

The plugin walks each path recursively for `.m4b` and `.mp3` files. It
only reads — never writes. The library on disk is canonical.

## 3. Scanning

Three trigger paths:

- **Admin button** — POST to `/admin/scan` from the audiobooks portal
  admin UI ("Scan now"). Returns immediately with the `scan_event_id`.
- **Scheduled** — every 6 hours via the SDK's `scheduled_task.v1`. No
  config required.
- **Startup** — on first Configure after install, the plugin does not
  auto-scan. Run the admin trigger once to populate the library.

Progress + history: `GET /admin/scan/status` returns the most recent 50
`scan_event` rows.

## 4. Standalone HTTP port (optional)

When the operator wants mobile clients to stream M4B/MP3 byte ranges
directly from this plugin instead of through the audiobooks portal:

1. **Generate a shared secret** via the audiobooks portal admin page
   ("Generate streaming secret"). Copy the 32-byte base64 value.
2. **Set `standalone_http_listen`** on this plugin (e.g. `:7879` or
   `127.0.0.1:7879`).
3. **Set `stream_signing_secret`** to the secret from step 1.
4. **Set `cdn_signing_secret`** on the audiobooks portal to the same
   secret, and **`cdn_hostname`** to the public hostname (e.g.
   `audiobooks-cdn.example.com`).
5. **DNS**: `audiobooks-cdn.example.com` CNAME to the same host as
   `abs.example.com`.
6. **Reverse proxy**: terminate TLS for `audiobooks-cdn.example.com`,
   forward to `127.0.0.1:7879`.
7. **Restart both plugins**.

The mobile client points at `abs.example.com` unchanged; the portal
issues redirect URLs to `audiobooks-cdn.example.com/api/v1/file/...` and
the client follows them transparently.

### Rotating the streaming secret

Manual: generate a new secret in the audiobooks portal admin UI, paste
into both plugin configs, restart both plugins. In-flight stream tokens
expire within 5 minutes regardless.

## 5. Metadata enrichment

The plugin enriches audiobooks with external metadata via a Postgres-backed
queue. The `metadata_enrichment_worker` scheduled task drains the queue once
per minute (cron `* * * * *`).

### Enrichment sources

Seven sources are bundled. Control which are active with `metadata_sources_enabled`
(defaults to all). The enrichment worker uses the single source named in
`metadata_scan_source` (default `audnexus`); gRPC `Search` queries all enabled
sources in parallel.

### Triggering backfill

To re-enrich all audiobooks (e.g. after adding a new source or changing region):

```
POST /admin/metadata/backfill
```

This enqueues every unenriched audiobook. The worker drains it at its next
scheduled minute.

### Rate limiting and caching

- `metadata_rate_limit_rps` (default `5`) limits outbound requests per source.
- `metadata_cache_ttl_days` (default `30`) controls how long a positive result
  is cached before re-fetching.
- `metadata_default_region` (default `"us"`) sets the ISO country code for
  region-aware sources.

### Inline enrichment

Set `scan_inline_enrich = true` to run enrichment synchronously after each
library scan finishes instead of queuing for the next worker run. Suitable for
small libraries; avoid on large ones.

## 6. Backups

The plugin's schema contains the catalog index + cover bytes + scan
history. The on-disk M4B/MP3 files are authoritative content — losing
the schema is recoverable via a rescan (durations / chapters re-extract
from the files). Cover bytes that came from sidecar files are also
recoverable; embedded covers re-extract from each file's tag.

A periodic `pg_dump --schema=local_audiobooks` is sufficient if you want to
avoid the rescan cost after a DR event.
