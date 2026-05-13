package sources

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newAudiobookCoversFake(t *testing.T) (*httptest.Server, *AudiobookCovers) {
	t.Helper()
	cover := loadFixture(t, "audiobookcovers_cover.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cover/by_book/B08G9PRS1K":
			w.Write(cover)
		default:
			w.WriteHeader(404)
		}
	}))
	a := NewAudiobookCoversAt(srv.URL, "test")
	a.http.Client = srv.Client()
	return srv, a
}

func TestAudiobookCovers_GetByASIN(t *testing.T) {
	srv, a := newAudiobookCoversFake(t)
	defer srv.Close()

	c, err := a.Get(context.Background(), "B08G9PRS1K", "us")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("nil candidate")
	}
	// Cover-only source: CoverURL must be populated.
	if c.CoverURL == "" {
		t.Error("CoverURL is empty")
	}
	// ASIN must echo the input.
	if c.ASIN != "B08G9PRS1K" {
		t.Errorf("ASIN %q", c.ASIN)
	}
	if c.ExternalID != "B08G9PRS1K" {
		t.Errorf("ExternalID %q", c.ExternalID)
	}
	if c.Source != audiobookcoversID {
		t.Errorf("Source %q", c.Source)
	}
	// Cover-only: title, authors, narrators must be empty.
	if c.Title != "" {
		t.Errorf("Title should be empty, got %q", c.Title)
	}
	if len(c.Authors) != 0 {
		t.Errorf("Authors should be empty, got %v", c.Authors)
	}
	if len(c.Narrators) != 0 {
		t.Errorf("Narrators should be empty, got %v", c.Narrators)
	}
	// Raw must be populated.
	if len(c.Raw) == 0 {
		t.Error("Raw is empty")
	}
}

func TestAudiobookCovers_GetMissing(t *testing.T) {
	srv, a := newAudiobookCoversFake(t)
	defer srv.Close()

	c, err := a.Get(context.Background(), "B0MISSING0", "us")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got err=%v c=%v", err, c)
	}
	if c != nil {
		t.Errorf("expected nil candidate, got %+v", c)
	}
}

func TestAudiobookCovers_SearchByASIN(t *testing.T) {
	srv, a := newAudiobookCoversFake(t)
	defer srv.Close()

	cs, err := a.Search(context.Background(), "B08G9PRS1K", "us")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) < 1 {
		t.Fatalf("expected ≥1 candidate, got %d", len(cs))
	}
	if cs[0].CoverURL == "" {
		t.Error("CoverURL is empty in search result")
	}
}

func TestAudiobookCovers_SearchByTextReturnsEmpty(t *testing.T) {
	srv, a := newAudiobookCoversFake(t)
	defer srv.Close()

	cs, err := a.Search(context.Background(), "Project Hail Mary", "us")
	if err != nil {
		t.Fatal(err)
	}
	if cs != nil {
		t.Errorf("expected nil slice for text query, got %v", cs)
	}
}
