package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestAdminHomeIncludesOperationalSectionsAndMountGuidance(t *testing.T) {
	h := New(Deps{}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/admin?theme=midnight-cinema", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`data-tab-target="libraries"`,
		`data-tab-target="scans"`,
		`data-tab-target="metadata"`,
		`data-tab-target="diagnostics"`,
		`Paths are validated inside the plugin container`,
		`id="diagnostics"`,
		`data-theme="midnight-cinema"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("admin home missing %q", want)
		}
	}
}

func TestAdminFilesystemBrowseListsDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Books"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "not-a-directory.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := New(Deps{}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/admin/filesystem/browse?path="+root, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"path":"`+root+`"`) {
		t.Fatalf("body missing current path: %s", body)
	}
	if !strings.Contains(body, `"name":"Books"`) {
		t.Fatalf("body missing child directory: %s", body)
	}
	if strings.Contains(body, "not-a-directory.txt") {
		t.Fatalf("body included non-directory entry: %s", body)
	}
}

func TestAdminFilesystemBrowseRejectsRelativePath(t *testing.T) {
	h := New(Deps{}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/admin/filesystem/browse?path=relative", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"bad_request"`) {
		t.Fatalf("body missing bad_request code: %s", rec.Body.String())
	}
}
