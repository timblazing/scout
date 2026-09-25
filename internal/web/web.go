// Package web serves the dashboard: a single-page app built from web/ by Vite
// and embedded into the binary from internal/web/dist.
package web

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// The all: prefix keeps dist/.gitkeep, so the package compiles before the UI
// has been built.
//
//go:embed all:dist
var distFS embed.FS

// notBuilt is served in place of the dashboard when the binary was compiled
// without running the frontend build first.
const notBuilt = `<!doctype html><html><head><meta charset="utf-8"><title>Scout</title></head>
<body style="background:#0a0a0a;color:#a1a1aa;font:14px system-ui;display:grid;place-items:center;height:100vh;margin:0">
<p>The dashboard has not been built. Run <code style="color:#fafafa">make web</code> and rebuild.</p></body></html>`

// Handler serves the embedded dashboard.
type Handler struct {
	files fs.FS
	etags map[string]string
}

// NewHandler indexes the embedded files, hashing each once so requests can be
// answered with an ETag without re-reading anything.
func NewHandler() *Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err) // the directory is embedded at compile time
	}
	return newHandler(sub)
}

func newHandler(files fs.FS) *Handler {
	h := &Handler{files: files, etags: make(map[string]string)}
	fs.WalkDir(files, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(files, p)
		if err != nil {
			return nil
		}
		sum := sha256.Sum256(data)
		h.etags[p] = `"` + hex.EncodeToString(sum[:8]) + `"`
		return nil
	})
	return h
}

// ServeHTTP serves a built asset, or index.html for any other path so the app
// can own its routing.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" || name == ".gitkeep" {
		name = "index.html"
	}

	data, err := fs.ReadFile(h.files, name)
	if err != nil {
		// A missing file with an extension is a genuine 404; anything else is
		// a client-side route.
		if name != "index.html" && path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		name = "index.html"
		if data, err = fs.ReadFile(h.files, name); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(notBuilt))
			return
		}
	}

	if strings.HasPrefix(name, "assets/") {
		// Vite puts a content hash in every asset filename, so these never
		// change under the same URL.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		// index.html and other unhashed files revalidate on every load so an
		// update is picked up immediately; the ETag keeps that to a 304.
		w.Header().Set("Cache-Control", "no-cache")
	}
	if etag, ok := h.etags[name]; ok {
		w.Header().Set("Etag", etag)
	}

	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}
