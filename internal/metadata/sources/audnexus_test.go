package sources

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func newAudnexusFake(t *testing.T) (*httptest.Server, *Audnexus) {
	book := loadFixture(t, "audnexus_book.json")
	chapters := loadFixture(t, "audnexus_chapters.json")
	author := loadFixture(t, "audnexus_author.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/books/") && strings.HasSuffix(r.URL.Path, "/chapters"):
			w.Write(chapters)
		case r.URL.Path == "/books/B08G9PRS1K":
			w.Write(book)
		case r.URL.Path == "/books/B0MISSING0":
			w.WriteHeader(404)
		case r.URL.Path == "/authors":
			w.Write(author)
		default:
			w.WriteHeader(404)
		}
	}))
	a := NewAudnexusAt(srv.URL, "test")
	a.http.Client = srv.Client()
	return srv, a
}

func TestAudnexus_GetByASIN(t *testing.T) {
	srv, a := newAudnexusFake(t)
	defer srv.Close()
	c, err := a.Get(context.Background(), "B08G9PRS1K", "us")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("nil candidate")
	}
	if c.Title != "Project Hail Mary" {
		t.Errorf("title %q", c.Title)
	}
	if c.ASIN != "B08G9PRS1K" {
		t.Errorf("asin %q", c.ASIN)
	}
	if c.Source != "audnexus" {
		t.Errorf("source %q", c.Source)
	}
	if c.RuntimeMin != 970 {
		t.Errorf("runtime %d", c.RuntimeMin)
	}
	if c.Series != "Hail Mary" || c.SeriesPos != "1" {
		t.Errorf("series %q %q", c.Series, c.SeriesPos)
	}
	if len(c.Authors) != 1 || c.Authors[0] != "Andy Weir" {
		t.Errorf("authors %v", c.Authors)
	}
	if len(c.Narrators) != 1 || c.Narrators[0] != "Ray Porter" {
		t.Errorf("narrators %v", c.Narrators)
	}
	if len(c.Chapters) != 2 {
		t.Errorf("chapters %d", len(c.Chapters))
	}
	if c.Chapters[1].StartMS != 300000 || c.Chapters[1].EndMS != 550000 {
		t.Errorf("chapter[1] = %+v", c.Chapters[1])
	}
}

func TestAudnexus_GetMissing(t *testing.T) {
	srv, a := newAudnexusFake(t)
	defer srv.Close()
	c, err := a.Get(context.Background(), "B0MISSING0", "us")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if c != nil {
		t.Errorf("expected nil candidate")
	}
}

func TestAudnexus_SearchByASIN(t *testing.T) {
	srv, a := newAudnexusFake(t)
	defer srv.Close()
	cs, err := a.Search(context.Background(), "B08G9PRS1K", "us")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 {
		t.Fatalf("got %d candidates", len(cs))
	}
}

func TestAudnexus_SearchByText(t *testing.T) {
	srv, a := newAudnexusFake(t)
	defer srv.Close()
	cs, err := a.Search(context.Background(), "Project Hail Mary", "us")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 {
		t.Fatalf("got %d candidates", len(cs))
	}
	if cs[0].Title != "Project Hail Mary" {
		t.Errorf("title %q", cs[0].Title)
	}
}

func TestAudnexus_NotAnASIN(t *testing.T) {
	srv, a := newAudnexusFake(t)
	defer srv.Close()
	c, err := a.Get(context.Background(), "not-an-asin", "us")
	if err != nil {
		t.Fatal(err)
	}
	if c != nil {
		t.Errorf("expected nil for non-ASIN")
	}
}
