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
