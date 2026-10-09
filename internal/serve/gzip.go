package serve

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// compressed sends text — the model, a scene, the app's own scripts — gzipped to a browser that
// asks for it. A model of a large repository is megabytes of repeated paths, and a tenth of that
// compressed. Images and ranged requests pass through untouched.
func compressed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipWriter{ResponseWriter: w}
		defer gw.close()
		next.ServeHTTP(gw, r)
	})
}

type gzipWriter struct {
	http.ResponseWriter
	zip     *gzip.Writer
	decided bool
}

func (g *gzipWriter) WriteHeader(status int) {
	if !g.decided {
		g.decided = true
		h := g.Header()
		h.Add("Vary", "Accept-Encoding")
		if status != http.StatusNoContent && status != http.StatusNotModified && h.Get("Content-Encoding") == "" && textual(h.Get("Content-Type")) {
			h.Set("Content-Encoding", "gzip")
			h.Del("Content-Length")
			g.zip = gzip.NewWriter(g.ResponseWriter)
		}
	}
	g.ResponseWriter.WriteHeader(status)
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	if !g.decided {
		if g.Header().Get("Content-Type") == "" {
			g.Header().Set("Content-Type", http.DetectContentType(b))
		}
		g.WriteHeader(http.StatusOK)
	}
	if g.zip != nil {
		return g.zip.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

func (g *gzipWriter) close() {
	if g.zip != nil {
		g.zip.Close()
	}
}

func textual(contentType string) bool {
	t, _, _ := strings.Cut(contentType, ";")
	t = strings.TrimSpace(t)
	return strings.HasPrefix(t, "text/") || t == "application/json" || t == "application/javascript" || t == "image/svg+xml"
}
