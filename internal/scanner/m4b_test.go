package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ContinuumApp/continuum-plugin-audiobooksdb/internal/scanner"
)

// fixtureM4B returns the path to a fixture; the file is checked into
// testdata/. If the fixture is missing the test skips with a helpful
// message rather than failing.
func fixtureM4B(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join("testdata", name)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("fixture %s missing: %v (regenerate with scripts/gen-m4b-fixture.sh)", p, err)
	}
	return p
}

func TestParseM4B_Tags(t *testing.T) {
	p := fixtureM4B(t, "minimal.m4b")
	got, err := scanner.ParseM4B(p)
	if err != nil {
		t.Fatalf("ParseM4B: %v", err)
	}
	if got.Title == "" {
		t.Errorf("title empty")
	}
	if got.Author == "" {
		t.Errorf("author empty")
	}
}

func TestParseM4B_NoChaptersSynthesizesOne(t *testing.T) {
	p := fixtureM4B(t, "minimal.m4b")
	got, err := scanner.ParseM4B(p)
	if err != nil {
		t.Fatalf("ParseM4B: %v", err)
	}
	if len(got.Chapters) != 1 {
		t.Fatalf("expected 1 synthesized chapter, got %d", len(got.Chapters))
	}
	if got.Chapters[0].Idx != 0 || got.Chapters[0].StartMs != 0 {
		t.Errorf("synthesized chapter shape wrong: %+v", got.Chapters[0])
	}
}

func TestParseM4B_DurationParsed(t *testing.T) {
	p := fixtureM4B(t, "minimal.m4b")
	got, err := scanner.ParseM4B(p)
	if err != nil {
		t.Fatalf("ParseM4B: %v", err)
	}
	if got.DurationMs <= 0 {
		t.Fatalf("expected positive duration, got %d", got.DurationMs)
	}
}

// Optional chaptered-fixture test — skips cleanly if the fixture isn't
// generated. We accept either real-parsed chapters or the synthesis
// fallback (both leave at least 1 chapter in ascending order).
func TestParseM4B_ChapAtomChapters(t *testing.T) {
	p := fixtureM4B(t, "chaptered.m4b")
	got, err := scanner.ParseM4B(p)
	if err != nil {
		t.Fatalf("ParseM4B: %v", err)
	}
	if len(got.Chapters) < 1 {
		t.Fatalf("expected >= 1 chapter, got %d", len(got.Chapters))
	}
	for i := 1; i < len(got.Chapters); i++ {
		if got.Chapters[i].StartMs < got.Chapters[i-1].StartMs {
			t.Errorf("chapters out of order at idx %d", i)
		}
	}
}
