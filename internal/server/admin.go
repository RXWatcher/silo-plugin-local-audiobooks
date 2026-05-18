package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ContinuumApp/continuum-plugin-local-audiobooks/internal/store"
)

func (s *Server) handleAdminListPaths(w http.ResponseWriter, r *http.Request) {
	paths, err := s.deps.Store.ListLibraryPaths(r.Context())
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": paths})
}

type addPathReq struct {
	Path string `json:"path"`
}

func (s *Server) handleAdminAddPath(w http.ResponseWriter, r *http.Request) {
	var body addPathReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		writeAdminError(w, http.StatusBadRequest, "invalid_input", "path required")
		return
	}
	info, err := os.Stat(body.Path)
	if err != nil || !info.IsDir() {
		writeAdminError(w, http.StatusBadRequest, "invalid_input", "path is not a readable directory")
		return
	}
	row, err := s.deps.Store.UpsertLibraryPath(r.Context(), body.Path)
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleAdminDeletePath(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid_input", "bad id")
		return
	}
	if err := s.deps.Store.DeleteLibraryPath(r.Context(), id); errors.Is(err, store.ErrNotFound) {
		writeAdminError(w, http.StatusNotFound, "not_found", "library path not found")
		return
	} else if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminScan(w http.ResponseWriter, r *http.Request) {
	if s.deps.Scan == nil {
		writeAdminError(w, http.StatusServiceUnavailable, "not_configured", "scan not configured")
		return
	}
	eventID, err := s.deps.Scan(r.Context())
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "scan_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"scan_event_id": eventID})
}

func (s *Server) handleAdminScanStatus(w http.ResponseWriter, r *http.Request) {
	events, err := s.deps.Store.ListRecentScanEvents(r.Context(), 50)
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events})
}

// handleMetadataBackfill enqueues a metadata_enrichment_job for every
// audiobook that has no existing enrichment job. Returns {"queued": <count>}.
func (s *Server) handleMetadataBackfill(w http.ResponseWriter, r *http.Request) {
	n, err := s.deps.Store.BulkEnqueueBackfill(r.Context())
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, "backfill_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int64{"queued": n})
}

func writeAdminError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
