// Package server builds the chi handler for /api/v1 and /admin routes.
package server

import (
	"context"
	"html"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ContinuumApp/continuum-plugin-local-audiobooks/internal/store"
)

// EnrichmentQueue is the surface the server needs from metadata.Queue.
// Declared as an interface to avoid an import cycle
// (metadata → store, store → server would cycle).
type EnrichmentQueue interface {
	Enqueue(ctx context.Context, audiobookID string) error
}

// Deps holds the handler's collaborators.
type Deps struct {
	Store        *store.Store
	StandaloneOn bool   // true when serving on the standalone listener
	StreamSecret []byte // shared HMAC for stream-token verification

	// Scan triggers a library scan. Returns the scan_event id. Multiple
	// concurrent calls de-duplicate to the same in-flight id. Nil-safe (the
	// admin handler returns 503 when Scan is nil).
	Scan func(context.Context) (int64, error)

	// MetadataQueue is optional. When non-nil, the /admin/metadata/backfill
	// endpoint enqueues enrichment jobs for all audiobooks lacking one.
	MetadataQueue EnrichmentQueue
}

// Server wraps the chi handler.
type Server struct {
	deps Deps
}

func New(d Deps) *Server { return &Server{deps: d} }

// Handler returns the chi router. When StandaloneOn is true, only file +
// cover endpoints answer; everything else returns 404. All standalone
// content endpoints require a valid stream-token query param (enforced in
// the handlers themselves — see T17).
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	if s.deps.StandaloneOn {
		r.Get("/api/v1/file/{id}", s.handleFileStandalone)
		r.Get("/api/v1/cover/{id}/{size}", s.handleCoverStandalone)
		r.NotFound(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":{"code":"not_allowed","message":"only file/cover are exposed on standalone listener"}}`, http.StatusNotFound)
		}))
		return r
	}
	r.Get("/api/v1/catalog", s.handleListCatalog)
	r.Get("/api/v1/catalog/libraries", s.handleListLibraries)
	r.Get("/api/v1/catalog/search", s.handleSearchCatalog)
	r.Get("/api/v1/catalog/{id}", s.handleGetCatalog)
	r.Get("/api/v1/browse/authors", s.handleBrowseAuthors)
	r.Get("/api/v1/browse/genres", s.handleBrowseGenres)
	r.Get("/api/v1/cover/{id}/{size}", s.handleCover)
	r.Get("/api/v1/file/{id}", s.handleFile)
	r.Get("/api/v1/requests/{externalId}", s.handleRequestsStub)
	r.Get("/admin", s.handleAdminHome)
	r.Get("/admin/", s.handleAdminHome)
	r.Post("/admin/scan", s.handleAdminScan)
	r.Get("/admin/scan/status", s.handleAdminScanStatus)
	r.Get("/admin/library-paths", s.handleAdminListPaths)
	r.Post("/admin/library-paths", s.handleAdminAddPath)
	r.Delete("/admin/library-paths/{id}", s.handleAdminDeletePath)
	r.Post("/admin/metadata/backfill", s.handleMetadataBackfill)
	return r
}

func (s *Server) handleAdminHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en" data-theme="` + adminTheme(r) + `">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Local Audiobooks</title><style>` + adminThemeCSS() + `</style></head>
<body>
<main class="shell">
<a class="back" href="/admin/plugins">&larr; Plugins</a>
<header><p class="eyebrow">Audiobook backend</p><h1>Local Audiobooks</h1><p>Local library scanning, metadata enrichment, covers, and file access for the Audiobooks portal.</p></header>
<section class="grid">
<article class="panel"><h2>Scan status</h2><div id="scan-status" class="muted">Loading...</div><div class="actions"><button id="scan" type="button">Run scan</button><button id="backfill" type="button">Queue metadata backfill</button></div></article>
<article class="panel"><h2>Library paths</h2><form id="path-form" class="row"><input id="path" placeholder="/mnt/audiobooks"><input id="name" placeholder="Display name"><button type="submit">Add</button></form><div id="paths" class="stack muted">Loading paths...</div></article>
</section>
<section class="panel"><h2>Operations checklist</h2><ul><li>Mount audiobook folders into the plugin container before adding paths.</li><li>Add at least one enabled path, then run a scan.</li><li>Add this backend as a presentation library in the Audiobooks portal.</li><li>Use metadata backfill after large imports or source changes.</li></ul></section>
</main>
<script>
const scanStatus=document.getElementById("scan-status"), pathsEl=document.getElementById("paths");
const hostToken=new URLSearchParams(location.search).get("token")||"";
function adminHeaders(extra){return Object.assign(hostToken?{Authorization:"Bearer "+hostToken}:{},extra||{})}
function esc(v){return String(v??"").replace(/[&<>"']/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]))}
async function json(url,init){const next=Object.assign({},init||{});next.headers=adminHeaders(next.headers);const r=await fetch(url,next);const d=await r.json().catch(()=>({}));if(!r.ok)throw new Error(d?.error?.message||d?.message||("HTTP "+r.status));return d}
async function loadStatus(){try{const d=await json("./admin/scan/status");scanStatus.innerHTML='<pre class="output">'+esc(JSON.stringify(d,null,2))+'</pre>'}catch(e){scanStatus.textContent=String(e)}}
async function loadPaths(){try{const d=await json("./admin/library-paths");const items=d.items||d.paths||[];pathsEl.innerHTML=items.length?items.map(p=>'<div class="path-row"><span><strong>'+esc(p.name||"Audiobooks")+'</strong><br><small>'+esc(p.path)+'</small></span><button data-id="'+esc(p.id)+'" type="button">Delete</button></div>').join(""):"No library paths configured.";pathsEl.querySelectorAll("button[data-id]").forEach(b=>b.addEventListener("click",async()=>{await json("./admin/library-paths/"+b.dataset.id,{method:"DELETE"});await loadPaths()}))}catch(e){pathsEl.textContent=String(e)}}
document.getElementById("scan").addEventListener("click",async()=>{scanStatus.textContent="Starting scan...";try{await json("./admin/scan",{method:"POST"});await loadStatus()}catch(e){scanStatus.textContent=String(e)}})
document.getElementById("backfill").addEventListener("click",async()=>{scanStatus.textContent="Queueing metadata backfill...";try{const d=await json("./admin/metadata/backfill",{method:"POST"});scanStatus.innerHTML='<pre class="output">'+esc(JSON.stringify(d,null,2))+'</pre>'}catch(e){scanStatus.textContent=String(e)}})
document.getElementById("path-form").addEventListener("submit",async e=>{e.preventDefault();try{await json("./admin/library-paths",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({path:document.getElementById("path").value,name:document.getElementById("name").value})});e.target.reset();await loadPaths()}catch(err){pathsEl.textContent=String(err)}})
loadStatus();loadPaths();
</script>
</body></html>`))
}

func adminTheme(r *http.Request) string {
	theme := r.Header.Get("X-Continuum-Theme")
	if theme == "" {
		theme = r.URL.Query().Get("theme")
	}
	if theme == "" {
		theme = "default"
	}
	return html.EscapeString(theme)
}

func adminThemeCSS() string {
	return `:root{--bg:#141417;--fg:#e8e8ec;--muted:#a1a1aa;--link:#93c5fd;--panel:#1c1c20;--border:#28282e;--input:#101014}[data-theme="cinema-light"]{--bg:#f7f3ed;--fg:#201c18;--muted:#756b60;--link:#9a3412;--panel:#fffaf3;--border:#ded1c0;--input:#fff}[data-theme="cobalt-studio"]{--bg:#101623;--fg:#eef4ff;--muted:#afc2e2;--link:#60a5fa;--panel:#172033;--border:#2d3f61;--input:#0d1422}[data-theme="oxblood-noir"]{--bg:#170b10;--fg:#fff1f4;--muted:#f0a6b7;--link:#fb7185;--panel:#241018;--border:#4a2230;--input:#12070b}[data-theme="evergreen-studio"]{--bg:#0d1712;--fg:#ecfdf3;--muted:#9bd6b4;--link:#6ee7b7;--panel:#14241b;--border:#2b4b39;--input:#08110d}body{font-family:system-ui,sans-serif;margin:0;line-height:1.5;background:var(--bg);color:var(--fg)}.shell{max-width:1120px;margin:0 auto;padding:28px}.back{color:var(--link);text-decoration:none}.eyebrow{color:var(--muted);text-transform:uppercase;font-size:12px;letter-spacing:.08em}h1{margin:.2rem 0}h2{font-size:16px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:16px}.panel{border:1px solid var(--border);background:var(--panel);border-radius:8px;padding:16px;margin-top:16px}.row{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,180px) auto;gap:8px}.actions{display:flex;gap:8px;margin-top:12px;flex-wrap:wrap}.stack>*+*{margin-top:8px}input{min-width:0;background:var(--input);color:var(--fg);border:1px solid var(--border);border-radius:6px;padding:9px}button{background:var(--link);border:0;border-radius:6px;padding:9px 12px;color:#08111f;font-weight:700}.muted{color:var(--muted)}.output{overflow:auto;max-height:340px;background:var(--input);border:1px solid var(--border);border-radius:6px;padding:10px;color:var(--fg)}.path-row{display:flex;align-items:center;justify-content:space-between;gap:10px;border:1px solid var(--border);border-radius:6px;padding:10px}.path-row small{color:var(--muted)}code{color:var(--link)}@media(max-width:760px){.row{grid-template-columns:1fr}}`
}
