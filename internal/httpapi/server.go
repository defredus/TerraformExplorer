package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"github.com/terraform-explorer/terraform-explorer/internal/app"
)

const maxRequestBody = 64 << 10

type Server struct {
	store  *app.Store
	assets fs.FS
	mux    *http.ServeMux
}

func New(store *app.Store, assets fs.FS) http.Handler {
	server := &Server{store: store, assets: assets, mux: http.NewServeMux()}
	server.routes()
	return securityHeaders(sameOriginOnly(server.mux))
}

func (server *Server) routes() {
	server.mux.HandleFunc("GET /api/v1/health", server.health)
	server.mux.HandleFunc("POST /api/v1/scans", server.createScan)
	server.mux.HandleFunc("GET /api/v1/scans/{id}", server.getScan)
	server.mux.HandleFunc("GET /api/v1/scans/{id}/source", server.getSource)
	server.mux.HandleFunc("GET /", server.static)
}

func (server *Server) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (server *Server) createScan(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.Path = strings.TrimSpace(input.Path)
	if input.Path == "" {
		writeError(writer, http.StatusBadRequest, "path_required", "A Terraform module directory is required.")
		return
	}

	snapshot, err := server.store.Scan(input.Path)
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, "scan_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusCreated, snapshot)
}

func (server *Server) getScan(writer http.ResponseWriter, request *http.Request) {
	snapshot, err := server.store.Get(request.PathValue("id"))
	if err != nil {
		if errors.Is(err, app.ErrSnapshotNotFound) {
			writeError(writer, http.StatusNotFound, "scan_not_found", err.Error())
			return
		}
		writeError(writer, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, snapshot)
}

func (server *Server) getSource(writer http.ResponseWriter, request *http.Request) {
	source, err := server.store.ReadSource(request.PathValue("id"), request.URL.Query().Get("file"))
	if err != nil {
		switch {
		case errors.Is(err, app.ErrSnapshotNotFound):
			writeError(writer, http.StatusNotFound, "scan_not_found", err.Error())
		case errors.Is(err, app.ErrInvalidSource):
			writeError(writer, http.StatusBadRequest, "invalid_source", err.Error())
		default:
			writeError(writer, http.StatusInternalServerError, "source_read_failed", err.Error())
		}
		return
	}
	writeJSON(writer, http.StatusOK, source)
}

func (server *Server) static(writer http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	if info, err := fs.Stat(server.assets, path); err == nil && !info.IsDir() {
		http.ServeFileFS(writer, request, server.assets, path)
		return
	}
	http.ServeFileFS(writer, request, server.assets, "index.html")
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, target any) error {
	request.Body = http.MaxBytesReader(writer, request.Body, maxRequestBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain one JSON object")
	}
	return nil
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func sameOriginOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" && request.Method != http.MethodGet && request.Method != http.MethodHead {
			parsed, err := url.Parse(origin)
			if err != nil || parsed.Host != request.Host || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				writeError(writer, http.StatusForbidden, "origin_rejected", "Cross-origin requests are not allowed by the local application.")
				return
			}
		}
		next.ServeHTTP(writer, request)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; worker-src 'self' blob:; connect-src 'self'")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(writer, request)
	})
}
