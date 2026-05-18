package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminErrorsUseJSONEnvelope(t *testing.T) {
	h := New(Deps{}).Handler()

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		status int
		code   string
	}{
		{
			name:   "bad add path request",
			method: http.MethodPost,
			path:   "/admin/library-paths",
			body:   `{}`,
			status: http.StatusBadRequest,
			code:   "invalid_input",
		},
		{
			name:   "bad delete id",
			method: http.MethodDelete,
			path:   "/admin/library-paths/not-a-number",
			status: http.StatusBadRequest,
			code:   "invalid_input",
		},
		{
			name:   "scan not configured",
			method: http.MethodPost,
			path:   "/admin/scan",
			status: http.StatusServiceUnavailable,
			code:   "not_configured",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tc.status, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
				t.Fatalf("content-type = %q, want json", got)
			}
			if !strings.Contains(rec.Body.String(), `"code":"`+tc.code+`"`) {
				t.Fatalf("body missing code %q: %s", tc.code, rec.Body.String())
			}
		})
	}
}
