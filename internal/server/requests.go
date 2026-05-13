package server

import "net/http"

func (s *Server) handleRequestsStub(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"error":"local backend has no concept of requests"}`))
}
