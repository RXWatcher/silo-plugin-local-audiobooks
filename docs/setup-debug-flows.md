# Local Audiobooks Setup, Debugging, And Flows

Plugin ID: `continuum.local-audiobooks`
Version documented: `0.2.0`

## Purpose

local filesystem audiobook backend for continuum.audiobooks.

## Runtime Dependencies

- Continuum plugin host
- Postgres schema for this plugin
- Mounted audiobook folders visible to the plugin runtime
- continuum.audiobooks for the user-facing portal

## Setup Checklist

1. Create schema and configure database_url.
2. Mount audiobook folders into the Continuum/plugin environment.
3. Configure library_paths as a JSON array of absolute paths.
4. Configure metadata sources and optional stream signing.
5. Run the library_scan scheduled task or wait for the schedule.
6. Map the backend into a presentation library in continuum.audiobooks.

## Configuration Reference

- `database_url`
- `library_paths`
- `standalone_http_listen`
- `stream_signing_secret`
- `metadata_sources_enabled`
- `metadata_default_region`
- `metadata_cache_ttl_days`
- `metadata_rate_limit_rps`
- `scan_inline_enrich`
- `metadata_scan_source`

Use the plugin manifest/admin form as the source of truth for field validation and defaults. Keep database credentials scoped to the plugin schema unless a plugin explicitly needs read access to Continuum core tables.

## Exposed Routes

- `* /api/v1/* [authenticated]`
- `* /admin/* [admin]`

## Capabilities

- `audiobook_backend.v1 (local_audiobooks) - Scans local audiobook libraries and exposes them to the Audiobooks portal.`
- `http_routes.v1 (api) - Catalog + admin HTTP routes`
- `scheduled_task.v1 (library_scan) - Periodic library rescan`
- `metadata_provider.v1 (local_audiobooks_meta) - Searches Audnexus, AudiMeta, iTunes, Storytel, BookBeat, Audioteka, and Audiobookcovers.`
- `scheduled_task.v1 (metadata_enrichment_worker) - Metadata enrichment worker`

## Operational Flows

### Scan/catalog

1. The scheduled scanner walks library_paths.
2. Audio files, folders, chapters, covers, and metadata hints are stored in the plugin schema.
3. Audiobooks portal calls this backend for libraries/search/detail/cover data.

### Streaming

1. Portal requests playback for a local title.
2. The backend returns stream/range endpoints or signed URLs.
3. The client requests byte ranges; progress remains in the portal.

### Enrichment

1. Scanner or metadata worker queues metadata lookup.
2. Configured metadata providers are called with rate limits/cache.
3. Normalized metadata updates the backend catalog.

## How This Plugin Communicates

- Implements audiobook_backend.v1 for continuum.audiobooks.
- Does not own the portal UI or request table.
- May publish/import events consumed by the Audiobooks portal.

## Debugging Runbook

- If the catalog is empty, check path mounts from inside the plugin container and JSON syntax of library_paths.
- If scans are slow, reduce metadata sources or disable scan_inline_enrich.
- If streams fail, check file permissions and stream_signing_secret alignment with the portal.
- Use docs/operations.md for operational runbooks and scan/enrichment notes.

## Log And Health Checks

- Start with Continuum Admin -> Plugins and confirm the installation is enabled.
- Check the plugin process logs around startup for manifest loading, migration, and route registration.
- Check scheduled task logs when a workflow depends on polling or reconciliation.
- Confirm the plugin routes are reachable through Continuum using the access level shown above.
- For database-backed plugins, verify the configured role can connect, create/migrate tables in its schema, and read/write expected rows.

## Common Failure Patterns

- Wrong installation ID selected in a portal or router setting after reinstalling a plugin.
- Plugin database URL points at the public schema instead of the dedicated plugin schema.
- Reverse proxy forwards the SPA route but not `/api/*`, `/api/v1/*`, `/assets/*`, or provider-specific public routes.
- Network checks are run from the operator laptop instead of from the Continuum/plugin runtime network.
- Secrets are regenerated during restart, invalidating signed URLs, encrypted fields, or login state.

## Verification After Changes

1. Restart or reload the plugin installation.
2. Open the plugin route or admin page in Continuum.
3. Exercise the smallest workflow that crosses a plugin boundary.
4. Confirm both the source plugin and destination plugin record the same request/session/login identifier.
5. Leave the scheduled reconciler enough time to run, then confirm terminal state or a useful error.
