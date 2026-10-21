package httpapi

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/terraform-explorer/terraform-explorer/internal/app"
)

func TestCreateScanAndReadSource(t *testing.T) {
	handler := New(app.NewStore(), testAssets())
	directory, err := filepath.Abs(filepath.Join("..", "..", "testdata", "simple-module"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"path": directory})
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/scans", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create scan status = %d, body = %s", response.Code, response.Body.String())
	}

	var snapshot app.Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Graph.Nodes) == 0 || len(snapshot.Graph.Edges) == 0 {
		t.Fatalf("snapshot graph is empty: %#v", snapshot.Graph)
	}

	sourceRequest := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/v1/scans/"+snapshot.ID+"/source?file=network.tf", nil)
	sourceResponse := httptest.NewRecorder()
	handler.ServeHTTP(sourceResponse, sourceRequest)
	if sourceResponse.Code != http.StatusOK || !strings.Contains(sourceResponse.Body.String(), "aws_s3_bucket") {
		t.Fatalf("source status = %d, body = %s", sourceResponse.Code, sourceResponse.Body.String())
	}
}

func TestCreateScanRejectsCrossOriginRequest(t *testing.T) {
	handler := New(app.NewStore(), testAssets())
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/scans", strings.NewReader(`{"path":"."}`))
	request.Header.Set("Origin", "https://malicious.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestStaticFallback(t *testing.T) {
	handler := New(app.NewStore(), testAssets())
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/workspace/anything", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "test app") {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func testAssets() fs.FS {
	return fstest.MapFS{"index.html": {Data: []byte("<h1>test app</h1>")}}
}
