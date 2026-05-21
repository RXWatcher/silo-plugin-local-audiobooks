package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// handleStream serves /api/v1/stream/{book_id}/{file_idx}. Local audiobooks
// follow a single-file-per-book model, so file_idx must be 0. Any other
// index is reported as not-found so callers don't silently get the wrong
// track. The route is declared public on the host plugin proxy (manifest);
// auth is enforced here via the signed media token in ?token=.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	bookID := chi.URLParam(r, "book_id")
	idxStr := chi.URLParam(r, "file_idx")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		http.Error(w, "file_idx must be a non-negative integer", http.StatusBadRequest)
		return
	}
	if err := requireStreamToken(r, s.deps.StreamSecret, bookID, idx); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if idx != 0 {
		http.Error(w, "file index not found", http.StatusNotFound)
		return
	}
	s.serveBookFile(w, r, bookID)
}

func (s *Server) handleStreamStandalone(w http.ResponseWriter, r *http.Request) {
	bookID := chi.URLParam(r, "book_id")
	idxStr := chi.URLParam(r, "file_idx")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		http.Error(w, "file_idx must be a non-negative integer", http.StatusBadRequest)
		return
	}
	if err := requireStreamToken(r, s.deps.StreamSecret, bookID, idx); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if idx != 0 {
		http.Error(w, "file index not found", http.StatusNotFound)
		return
	}
	s.serveBookFile(w, r, bookID)
}

func (s *Server) serveBookFile(w http.ResponseWriter, r *http.Request, bookID string) {
	book, err := s.deps.Store.GetAudiobook(r.Context(), bookID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	f, err := os.Open(book.Path)
	if err != nil {
		http.Error(w, "file not readable", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", contentTypeFor(book.Path))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("X-Stream-Source", "local-fs")
	http.ServeContent(w, r, book.Path, book.MTime, f)
}

func contentTypeFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".m4b", ".m4a":
		return "audio/mp4"
	case ".mp3":
		return "audio/mpeg"
	}
	return "application/octet-stream"
}
