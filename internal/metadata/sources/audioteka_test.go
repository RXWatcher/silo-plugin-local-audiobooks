package sources

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newAudiotekaFake(t *testing.T) (*httptest.Server, *Audioteka) {
	t.Helper()
	book := loadFixture(t, "audioteka_book.html")
	search := loadFixture(t, "audioteka_search.html")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/audiobook/wiedzmin-ostatnie-zyczenie":
			w.Write(book)
		case r.URL.Path == "/audiobook/does-not-exist":
			w.WriteHeader(404)
		case r.URL.Path == "/szukaj":
			w.Write(search)
		default:
			w.WriteHeader(404)
		}
	}))
	a := NewAudiotekaAt(srv.URL, "test")
	a.http.Client = srv.Client()
	return srv, a
}

func TestAudioteka_GetByID(t *testing.T) {
	srv, a := newAudiotekaFake(t)
	defer srv.Close()

	c, err := a.Get(context.Background(), "wiedzmin-ostatnie-zyczenie", "pl")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("nil candidate")
	}
	if c.Title != "Wiedźmin: Ostatnie życzenie" {
		t.Errorf("title = %q, want %q", c.Title, "Wiedźmin: Ostatnie życzenie")
	}
	if c.ExternalID == "" {
		t.Errorf("ExternalID is empty")
	}
	if c.Source != audiotekaID {
		t.Errorf("source = %q, want %q", c.Source, audiotekaID)
	}
	// PT9H27M = 9*60 + 27 = 567 minutes
	if c.RuntimeMin != 567 {
		t.Errorf("runtime_min = %d, want 567", c.RuntimeMin)
	}
	if len(c.Authors) == 0 || c.Authors[0] != "Andrzej Sapkowski" {
		t.Errorf("authors = %v, want [Andrzej Sapkowski]", c.Authors)
	}
	if len(c.Narrators) == 0 || c.Narrators[0] != "Jacek Rozenek" {
		t.Errorf("narrators = %v, want [Jacek Rozenek]", c.Narrators)
	}
	if c.ISBN != "9788375170177" {
		t.Errorf("isbn = %q, want %q", c.ISBN, "9788375170177")
	}
	if c.Raw == nil {
		t.Errorf("Raw is nil")
	}
}

func TestAudioteka_GetMissing(t *testing.T) {
	srv, a := newAudiotekaFake(t)
	defer srv.Close()

	c, err := a.Get(context.Background(), "does-not-exist", "pl")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if c != nil {
		t.Errorf("expected nil candidate, got %+v", c)
	}
}

func TestAudioteka_SearchByText(t *testing.T) {
	srv, a := newAudiotekaFake(t)
	defer srv.Close()

	cs, err := a.Search(context.Background(), "wiedźmin", "pl")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) < 1 {
		t.Fatalf("got %d candidates, want >=1", len(cs))
	}
	if cs[0].Title != "Wiedźmin: Ostatnie życzenie" {
		t.Errorf("first title = %q, want %q", cs[0].Title, "Wiedźmin: Ostatnie życzenie")
	}
	if cs[0].Source != audiotekaID {
		t.Errorf("source = %q, want %q", cs[0].Source, audiotekaID)
	}
	// duration 34020 seconds / 60 = 567 minutes
	if cs[0].RuntimeMin != 567 {
		t.Errorf("runtime_min = %d, want 567", cs[0].RuntimeMin)
	}
	if cs[0].Series != "Wiedźmin" {
		t.Errorf("series = %q, want %q", cs[0].Series, "Wiedźmin")
	}
	if cs[0].SeriesPos != "1" {
		t.Errorf("series_pos = %q, want %q", cs[0].SeriesPos, "1")
	}
}

func TestAudioteka_ASINReturnsNil(t *testing.T) {
	srv, a := newAudiotekaFake(t)
	defer srv.Close()

	c, err := a.Get(context.Background(), "B08G9PRS1K", "pl")
	if err != nil {
		t.Errorf("Get ASIN: unexpected error: %v", err)
	}
	if c != nil {
		t.Errorf("Get ASIN: expected nil candidate, got %+v", c)
	}

	cs, err := a.Search(context.Background(), "B08G9PRS1K", "pl")
	if err != nil {
		t.Errorf("Search ASIN: unexpected error: %v", err)
	}
	if cs != nil {
		t.Errorf("Search ASIN: expected nil, got %v", cs)
	}
}
