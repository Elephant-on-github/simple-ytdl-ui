package api

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

// distFS holds the built single page application. The real files are copied
// into this directory when the image is built.
//
//go:embed all:dist
var distFS embed.FS

// StaticHandler serves the single page application. When webDir is set that
// directory is used, otherwise the embedded copy is served. Unknown paths
// fall back to index.html so deep links keep working after a reload.
func StaticHandler(webDir string) http.Handler {
	if webDir != "" {
		return spaHandler{fs: os.DirFS(webDir)}
	}
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// The embed directive guarantees the directory exists.
		panic(err)
	}
	return spaHandler{fs: sub}
}

// spaHandler serves a file system with an index.html fallback.
type spaHandler struct {
	fs fs.FS
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name == "" || name == "." || strings.HasSuffix(name, "/") {
		h.serveIndex(w, r)
		return
	}
	h.serveFile(w, r, name)
}

func (h spaHandler) serveFile(w http.ResponseWriter, r *http.Request, name string) {
	file, err := h.fs.Open(name)
	if err != nil {
		h.serveIndex(w, r)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		h.serveIndex(w, r)
		return
	}
	seeker, ok := file.(io.ReadSeeker)
	if !ok {
		h.serveIndex(w, r)
		return
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), seeker)
}

func (h spaHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	file, err := h.fs.Open("index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	seeker, ok := file.(io.ReadSeeker)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, "index.html", info.ModTime(), seeker)
}
