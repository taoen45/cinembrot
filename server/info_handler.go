package server

import (
	"net/http"
	"os"
	"path/filepath"
)

// HandleInfoDomain serves the official information landing page (cinembrot.my.id)
// similar to filmapik.info with official active domain notice and SEO articles
func (s *Server) HandleInfoDomain(w http.ResponseWriter, r *http.Request) {
	infoPath := filepath.Join("public", "info", "index.html")
	if _, err := os.Stat(infoPath); err == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=600")
		http.ServeFile(w, r, infoPath)
		return
	}

	// Fallback minimal HTML if file not found
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html><html><head><title>CINEMBROT Info</title></head><body><h1>Domain Resmi CINEMBROT: <a href="https://cinembrot.web.id">cinembrot.web.id</a></h1></body></html>`))
}
