package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"terax/internal/workspace"
)

// localFileHandler serves disk files for the `/local-file/<path>` URLs that
// `convertFileSrc` produces (image / PDF / spreadsheet previews). Wails' asset
// server only serves the embedded `frontend/dist` assets, so every local-file
// request falls through to this Handler.
//
// Only paths inside an authorized workspace are served; everything else gets
// a 404/403 so the webview can't read arbitrary disk files.
func localFileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "/local-file/"
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		raw := strings.TrimPrefix(r.URL.Path, prefix)
		path, err := url.PathUnescape(raw)
		if err != nil || path == "" {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		if !workspace.IsAuthorized(path) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		f, err := os.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	})
}