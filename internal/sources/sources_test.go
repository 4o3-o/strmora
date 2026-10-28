package sources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalSource(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "movies")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(subDir, "test.mkv")
	if err := os.WriteFile(filePath, []byte("fake-video-bytes"), 0644); err != nil {
		t.Fatal(err)
	}

	src, err := NewLocal(Config{
		ID:       "local-1",
		RootPath: dir,
		ReadOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := src.Ping(ctx); err != nil {
		t.Fatalf("local ping failed: %v", err)
	}

	caps := src.Capabilities()
	if !caps.ReadOnly {
		t.Fatalf("expected ReadOnly=true")
	}
	if caps.SupportsBatchMove {
		t.Fatalf("expected SupportsBatchMove=false for local")
	}

	// List directory
	entries, err := src.List(ctx, "movies")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "test.mkv" || entries[0].IsDir {
		t.Fatalf("unexpected entries: %+v", entries)
	}

	// Read file
	rc, stat, err := src.Open(ctx, "movies/test.mkv")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer rc.Close()
	content, _ := io.ReadAll(rc)
	if string(content) != "fake-video-bytes" || stat.Size != int64(len("fake-video-bytes")) {
		t.Fatalf("unexpected content: %s", content)
	}
}

func TestOpenListSource(t *testing.T) {
	// Mock OpenList server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/api/fs/list":
			var req struct {
				Path string `json:"path"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.Path == "/115/movies" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"code": 200,
					"data": map[string]any{
						"content": []map[string]any{
							{"name": "one-punch.mkv", "size": 1024, "is_dir": false, "modified": time.Now().Format(time.RFC3339)},
							{"name": "Season 1", "size": 0, "is_dir": true, "modified": time.Now().Format(time.RFC3339)},
						},
						"total": 2,
					},
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		case "/d/115/movies/one-punch.mkv":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("mock-openlist-file"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	src, err := NewOpenList(Config{
		ID:       "openlist-1",
		Endpoint: ts.URL,
		Token:    "test-token",
		RootPath: "/115/movies",
		ReadOnly: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	caps := src.Capabilities()
	if !caps.SupportsBatchMove || !caps.SupportsBatchRename {
		t.Fatalf("OpenList should declare batch move & rename support: %+v", caps)
	}

	entries, err := src.List(ctx, "")
	if err != nil {
		t.Fatalf("openlist list failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	rc, _, err := src.Open(ctx, "one-punch.mkv")
	if err != nil {
		t.Fatalf("openlist open failed: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "mock-openlist-file" {
		t.Fatalf("unexpected content: %s", string(data))
	}
}

func TestWebDAVSource(t *testing.T) {
	// Mock WebDAV PROPFIND
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "davuser" || p != "davpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == "PROPFIND" {
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			// Return self and one child
			w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:">
  <D:response>
    <D:href>/dav/media/</D:href>
    <D:propstat>
      <D:prop>
        <D:resourcetype><D:collection/></D:resourcetype>
      </D:prop>
      <D:status>HTTP/1.1 200 OK</D:status>
    </D:propstat>
  </D:response>
  <D:response>
    <D:href>/dav/media/video.mp4</D:href>
    <D:propstat>
      <D:prop>
        <D:resourcetype/>
        <D:getcontentlength>2048</D:getcontentlength>
      </D:prop>
      <D:status>HTTP/1.1 200 OK</D:status>
    </D:propstat>
  </D:response>
</D:multistatus>`))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/dav/media/video.mp4" {
			w.Write([]byte("webdav-data"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	src, err := NewWebDAV(Config{
		ID:       "webdav-1",
		Endpoint: ts.URL + "/dav/media",
		Username: "davuser",
		Password: "davpass",
		ReadOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	caps := src.Capabilities()
	if caps.SupportsBatchMove {
		t.Fatalf("Standard WebDAV cannot support multi-file single-request batch move")
	}

	entries, err := src.List(ctx, "")
	if err != nil {
		t.Fatalf("webdav list failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "video.mp4" || entries[0].Size != 2048 {
		t.Fatalf("unexpected webdav entries: %+v", entries)
	}

	rc, _, err := src.Open(ctx, "video.mp4")
	if err != nil {
		t.Fatalf("webdav open failed: %v", err)
	}
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	if string(b) != "webdav-data" {
		t.Fatalf("unexpected content: %s", string(b))
	}
}
