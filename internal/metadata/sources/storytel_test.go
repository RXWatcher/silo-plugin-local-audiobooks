package sources

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newStorytelFake(t *testing.T) (*httptest.Server, *Storytel) {
	t.Helper()
	book := loadFixture(t, "storytel_book.json")
	search := loadFixture(t, "storytel_search.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/books/project-hail-mary-12345":
			w.Write(book)
		case r.URL.Path == "/books/does-not-exist-99999":
			w.WriteHeader(404)
		case r.URL.Path == "/search":
			w.Write(search)
		default:
			w.WriteHeader(404)
		}
	}))
	s := NewStorytelAt(srv.URL, "test")
	s.http.Client = srv.Client()
	return srv, s
}

func TestStorytel_GetByID(t *testing.T) {
	srv, s := newStorytelFake(t)
	defer srv.Close()

	c, err := s.Get(context.Background(), "project-hail-mary-12345", "us")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("nil candidate")
	}
	if c.Title != "Project Hail Mary" {
		t.Errorf("title = %q, want %q", c.Title, "Project Hail Mary")
	}
	if c.Source != storytelID {
		t.Errorf("source = %q, want %q", c.Source, storytelID)
	}
	if c.ExternalID == "" {
		t.Errorf("ExternalID is empty")
	}
	if len(c.Authors) == 0 || c.Authors[0] != "Andy Weir" {
		t.Errorf("authors = %v, want [Andy Weir]", c.Authors)
	}
	if len(c.Narrators) == 0 || c.Narrators[0] != "Ray Porter" {
		t.Errorf("narrators = %v, want [Ray Porter]", c.Narrators)
	}
	if c.Language != "en" {
		t.Errorf("language = %q, want %q", c.Language, "en")
	}
	if c.PublishedAt != "2021-05-04" {
		t.Errorf("published_at = %q, want %q", c.PublishedAt, "2021-05-04")
	}
	// PT16H11M = 971 minutes
	if c.RuntimeMin == 0 {
		t.Errorf("runtime_min is 0, want >0")
	}
	if c.Raw == nil {
		t.Errorf("Raw is nil")
	}
}

func TestStorytel_GetMissing(t *testing.T) {
	srv, s := newStorytelFake(t)
	defer srv.Close()

	c, err := s.Get(context.Background(), "does-not-exist-99999", "us")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if c != nil {
		t.Errorf("expected nil candidate, got %+v", c)
	}
}

func TestStorytel_SearchByText(t *testing.T) {
	srv, s := newStorytelFake(t)
	defer srv.Close()

	cs, err := s.Search(context.Background(), "hail mary", "us")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) < 1 {
		t.Fatalf("got %d candidates, want ≥1", len(cs))
	}
	if cs[0].Title != "Project Hail Mary" {
		t.Errorf("first title = %q, want %q", cs[0].Title, "Project Hail Mary")
	}
	if cs[0].Source != storytelID {
		t.Errorf("source = %q, want %q", cs[0].Source, storytelID)
	}
	// Check series mapping
	if cs[0].Series != "Hail Mary" {
		t.Errorf("series = %q, want %q", cs[0].Series, "Hail Mary")
	}
	if cs[0].SeriesPos != "1" {
		t.Errorf("series_pos = %q, want %q", cs[0].SeriesPos, "1")
	}
}

func TestStorytel_ASINReturnsNil(t *testing.T) {
	srv, s := newStorytelFake(t)
	defer srv.Close()

	c, err := s.Get(context.Background(), "B08G9PRS1K", "us")
	if err != nil {
		t.Errorf("Get ASIN: unexpected error: %v", err)
	}
	if c != nil {
		t.Errorf("Get ASIN: expected nil candidate, got %+v", c)
	}
}

func TestStorytel_RegionURL(t *testing.T) {
	// Use a production-baseURL instance so storytelHostFor uses the switch.
	s := NewStorytel("test")

	cases := []struct {
		region string
		want   string
	}{
		{"uk", "https://www.storytel.co.uk"},
		{"us", "https://www.storytel.com"},
		{"", "https://www.storytel.com"},
		{"de", "https://www.storytel.de"},
	}
	for _, tc := range cases {
		got := s.storytelHostFor(tc.region)
		if got != tc.want {
			t.Errorf("storytelHostFor(%q) = %q, want %q", tc.region, got, tc.want)
		}
	}
}
