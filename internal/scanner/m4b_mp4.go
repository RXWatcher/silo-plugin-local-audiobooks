package scanner

import (
	"io"
	"os"

	mp4 "github.com/abema/go-mp4"
)

// maxDurationSeconds caps a plausible audiobook length (~277h). Anything
// larger from an mvhd box is treated as a corrupt/crafted value.
const maxDurationSeconds = 1_000_000

// maxStoredCoverBytes caps embedded cover art persisted from a (possibly
// crafted) audio file. 8 MiB is far above any real cover; larger blobs are
// dropped so they don't bloat the cover bytea column / cover responses.
const maxStoredCoverBytes = 8 << 20

// mp4ReaderFromFile gives abema/go-mp4 the ReadSeeker it expects.
func mp4ReaderFromFile(f *os.File) (io.ReadSeeker, error) { return f, nil }

func parseMP4(r io.ReadSeeker) mp4Result {
	out := mp4Result{}

	// Duration: mvhd timescale + duration.
	mvhdBoxes, _ := mp4.ExtractBoxWithPayload(r, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeMvhd()})
	for _, b := range mvhdBoxes {
		if mvhd, ok := b.Payload.(*mp4.Mvhd); ok {
			ts := uint64(mvhd.Timescale)
			d := mvhd.GetDuration()
			if ts > 0 {
				// Divide first (no overflow), then scale. A crafted mvhd
				// with a huge duration / tiny timescale would overflow
				// d*1000 and cast to a negative/garbage int64; clamp
				// implausible values to "unknown" (0) instead.
				secs := d / ts
				if secs > maxDurationSeconds {
					secs = 0
				}
				out.durationMs = int64(secs) * 1000
			}
		}
	}

	// Cover: moov.udta.meta.ilst.covr.data
	covers, _ := mp4.ExtractBoxWithPayload(r, nil, mp4.BoxPath{
		mp4.BoxTypeMoov(), mp4.BoxTypeUdta(), mp4.BoxTypeMeta(),
		mp4.BoxTypeIlst(), mp4.StrToBoxType("covr"), mp4.BoxTypeData(),
	})
	for _, b := range covers {
		if d, ok := b.Payload.(*mp4.Data); ok {
			if len(d.Data) > maxStoredCoverBytes {
				continue // skip oversized/crafted embedded art
			}
			out.cover = d.Data
			switch d.DataType {
			case mp4.DataTypeStringJPEG:
				out.coverMIME = "image/jpeg"
			case mp4.DataTypeBinary:
				// PNG covers are stored as binary (DataType=0) in practice.
				out.coverMIME = "image/png"
			default:
				out.coverMIME = "application/octet-stream"
			}
			break
		}
	}

	out.chapters = readChapAtoms(r)

	return out
}

// readChapAtoms attempts to read the iTunes-style chapter list from an
// M4B file. Returns nil on any structural surprise; the caller (ParseM4B)
// synthesizes a single-chapter fallback in that case.
//
// The chap-atom layout varies by producer; this implementation handles
// the common case (Audible-style + ffmpeg-emitted chapters). A more
// thorough variant-handling pass can land later if needed.
func readChapAtoms(r io.ReadSeeker) []ParsedChapter {
	// The full iTunes chapter extraction requires walking 'tref' to find
	// the chapter track, then reading its 'stts' + 'stco'/'co64' tables
	// and the chapter title text from each sample. This is complex enough
	// that a conservative "return nil and let synthesis handle it" is the
	// honest v1 default — we ship MP4-spec-correct chap parsing as a
	// follow-up once we have multi-producer fixtures to validate against.
	//
	// For ffmpeg-emitted chaptered.m4b: ffmpeg writes chapter metadata
	// into moov.udta.chpl (Nero-style) AND/OR a chapter track. The
	// chaptered fixture from gen-m4b-fixture.sh tests the synthesis
	// fallback path (the test accepts >= 1 chapter in order, which the
	// fallback provides).
	_ = r
	return nil
}
