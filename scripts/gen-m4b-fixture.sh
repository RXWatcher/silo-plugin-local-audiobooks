#!/usr/bin/env bash
set -euo pipefail

OUT="internal/scanner/testdata/minimal.m4b"
mkdir -p "$(dirname "$OUT")"

# 1-second silence + ID3 metadata; smallest valid M4B we can craft.
# Requires ffmpeg locally — not a runtime dependency.
ffmpeg -y \
  -f lavfi -i anullsrc=channel_layout=mono:sample_rate=22050 -t 1 \
  -metadata title="Minimal Test Book" \
  -metadata artist="Test Narrator" \
  -metadata album_artist="Test Author" \
  -metadata album="Minimal Test Album" \
  -metadata date="2024" \
  -metadata genre="Science Fiction" \
  -c:a aac -b:a 16k -movflags +faststart \
  "$OUT"

echo "Generated: $OUT ($(du -h "$OUT" | cut -f1))"
