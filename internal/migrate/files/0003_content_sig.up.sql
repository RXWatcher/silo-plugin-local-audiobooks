-- content_sig decouples change-detection from the primary key. Previously the
-- scanner derived the audiobook id from (path,size,mtime), so any content edit
-- or even an mtime-only change (backup restore, fs copy, touch) minted a new
-- id: the old row was soft-deleted and a new one inserted, resetting metadata
-- enrichment and cover association and re-enqueuing enrichment. The id is now
-- stable per (library_path_id, path); content_sig carries the (size,mtime)
-- signature used only to skip unchanged files. Existing rows default to '' so
-- their first post-migration scan re-ingests once, then stabilises.
ALTER TABLE audiobook
  ADD COLUMN IF NOT EXISTS content_sig TEXT NOT NULL DEFAULT '';
