// Package scanner walks library paths and indexes M4B / MP3 files.
package scanner

import (
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/blake2b"
)

// StableID returns the deterministic id for an audio file. The id changes
// when path, size, or mtime changes. Same path + size + mtime always
// yields the same id. 128-bit blake2b, hex-encoded (32 chars).
func StableID(path string, size int64, mtime time.Time) string {
	h, _ := blake2b.New(16, nil) // 128-bit
	fmt.Fprintf(h, "%s\x00%d\x00%d", path, size, mtime.UnixNano())
	return hex.EncodeToString(h.Sum(nil))
}
