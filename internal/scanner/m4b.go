package scanner

import (
	"fmt"
	"os"

	"github.com/dhowden/tag"
)

// ParsedM4B is the file-level metadata extracted from an M4B by ParseM4B.
type ParsedM4B struct {
	Title       string
	Author      string
	Narrator    string
	Album       string
	Year        string
	Genre       string
	ISBN        string
	ASIN        string
	Description string

	// Filled in by future tasks:
	Chapters    []ParsedChapter
	CoverBytes  []byte
	CoverMIME   string
	CoverSource string // "embedded" or "sidecar"
	DurationMs  int64
}

// ParsedChapter is one chapter from the M4B chap atom (or synthesized).
type ParsedChapter struct {
	Idx     int
	Title   string
	StartMs int64
	EndMs   int64
}

// ParseM4B opens the file and returns its extracted metadata. The parser
// never modifies the file. Errors from missing tag atoms are tolerated and
// surface as empty fields, not failures — many M4Bs lack some fields.
func ParseM4B(path string) (*ParsedM4B, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	out := &ParsedM4B{}

	// Pass 1: dhowden/tag for ID3-style atoms.
	if m, err := tag.ReadFrom(f); err == nil {
		out.Title = m.Title()
		out.Album = m.Album()
		out.Genre = m.Genre()
		if y := m.Year(); y > 0 {
			out.Year = fmt.Sprintf("%d", y)
		}
		out.Description = m.Comment()
		// Heuristic: aART (album artist) is the author; ©ART (artist) is
		// the narrator when distinct.
		albumArtist := m.AlbumArtist()
		artist := m.Artist()
		switch {
		case albumArtist != "" && artist != "" && albumArtist != artist:
			out.Author = albumArtist
			out.Narrator = artist
		case albumArtist != "":
			out.Author = albumArtist
			out.Narrator = artist
		case artist != "":
			out.Author = artist
		}
		raw := m.Raw()
		if v, ok := raw["asin"].(string); ok {
			out.ASIN = v
		}
		if v, ok := raw["isbn"].(string); ok {
			out.ISBN = v
		}
	}

	// Pass 2 (Task 9): abema/go-mp4 for chapters + cover + duration.
	// Left for the next task.

	return out, nil
}
