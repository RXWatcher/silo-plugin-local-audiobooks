package sources

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newBookBeatFake(t *testing.T) (*httptest.Server, *BookBeat) {
	t.Helper()
	book := loadFixture(t, "bookbeat_book.html")
	search := loadFixture(t, "bookbeat_search.html")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/book/midnight-library-98765":
			w.Write(book)
		case r.URL.Path == "/book/does-not-exist-00000":
			w.WriteHeader(404)
		case r.URL.Path == "/bok/does-not-exist-00000":
			w.WriteHeader(404)
		case r.URL.Path == "/buch/does-not-exist-00000":
			w.WriteHeader(404)
		case r.URL.Path == "/boek/does-not-exist-00000":
			w.WriteHeader(404)
		case r.URL.Path == "/search":
			w.Write(search)
		default:
			w.WriteHeader(404)
		}
	}))
	b := NewBookBeatAt(srv.URL, "test")
	b.http.Client = srv.Client()
	return srv, b
}

func TestBookBeat_GetByID(t *testing.T) {
	srv, b := newBookBeatFake(t)
	defer srv.Close()

	c, err := b.Get(context.Background(), "midnight-library-98765", "se")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("nil candidate")
	}
	if c.Title != "The Midnight Library" {
		t.Errorf("title = %q, want %q", c.Title, "The Midnight Library")
	}
	if c.ExternalID == "" {
		t.Errorf("ExternalID is empty")
	}
	if c.Source != bookbeatID {
		t.Errorf("source = %q, want %q", c.Source, bookbeatID)
	}
	// PT8H49M = 8*60 + 49 = 529 minutes
	if c.RuntimeMin == 0 {
		t.Errorf("runtime_min is 0, want >0")
	}
	if len(c.Authors) == 0 || c.Authors[0] != "Matt Haig" {
		t.Errorf("authors = %v, want [Matt Haig]", c.Authors)
	}
	if c.Raw == nil {
		t.Errorf("Raw is nil")
	}
}

func TestBookBeat_GetMissing(t *testing.T) {
	srv, b := newBookBeatFake(t)
	defer srv.Close()

	c, err := b.Get(context.Background(), "does-not-exist-00000", "se")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if c != nil {
		t.Errorf("expected nil candidate, got %+v", c)
	}
}

func TestBookBeat_SearchByText(t *testing.T) {
	srv, b := newBookBeatFake(t)
	defer srv.Close()

	cs, err := b.Search(context.Background(), "midnight library", "se")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) < 1 {
		t.Fatalf("got %d candidates, want >=1", len(cs))
	}
	if cs[0].Title != "The Midnight Library" {
		t.Errorf("first title = %q, want %q", cs[0].Title, "The Midnight Library")
	}
	if cs[0].Source != bookbeatID {
		t.Errorf("source = %q, want %q", cs[0].Source, bookbeatID)
	}
}

func TestBookBeat_ASINReturnsNil(t *testing.T) {
	srv, b := newBookBeatFake(t)
	defer srv.Close()

	c, err := b.Get(context.Background(), "B08G9PRS1K", "se")
	if err != nil {
		t.Errorf("Get ASIN: unexpected error: %v", err)
	}
	if c != nil {
		t.Errorf("Get ASIN: expected nil candidate, got %+v", c)
	}
}
