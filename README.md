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

## What it does NOT do

- It does not modify your files. No tag rewrites, no moves, no renames.
- It does not fetch metadata from external services. That's the future
  metadata-agent system's job.
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
