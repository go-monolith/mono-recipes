package fileservice_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-monolith/mono"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"

	"github.com/example/file-upload-demo/modules/fileservice"
)

// The file-service module stores files in the fs-jetstream plugin, which
// needs the embedded NATS JetStream server, so monotest (NATS-free) cannot
// run it. This is the module's boundary test instead: a real application in
// strict mode, driven through its HTTP API, with a clean boundary report.

// freePort returns a free TCP port on localhost.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func startApp(t *testing.T) (mono.MonoApplication, string) {
	t.Helper()
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelError),
		mono.WithJetStreamStorageDir(t.TempDir()),
		// In-process NATS: no TCP port, so parallel test runs cannot collide.
		mono.WithNATSDontListen(),
		mono.WithNATSInProcessConn(),
		// Default (non-strict) mode on purpose: violations are recorded in
		// app.ModuleBoundaries() instead of rejected, so the report check below
		// is meaningful. main.go runs strict once this report is clean.
	)
	if err != nil {
		t.Fatal(err)
	}
	storage, err := fsjetstream.New(fsjetstream.Config{
		Buckets: []fsjetstream.BucketConfig{{Name: "files", Storage: fsjetstream.MemoryStorage}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.RegisterPlugin(storage, "storage"); err != nil {
		t.Fatal(err)
	}
	port := freePort(t)
	if err := app.Register(fileservice.NewModule(port, 10*1024*1024, app.Logger())); err != nil {
		t.Fatal(err)
	}
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Stop(context.Background()) })
	return app, fmt.Sprintf("http://127.0.0.1:%d", port)
}

// multipartBody builds a multipart form with one file per name in field.
func multipartBody(t *testing.T, field string, files map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for name, content := range files {
		part, err := w.CreateFormFile(field, name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, w.FormDataContentType()
}

func do(t *testing.T, method, url, contentType string, body io.Reader, wantStatus int) []byte {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s: status %d, want %d (body %s)", method, url, resp.StatusCode, wantStatus, data)
	}
	return data
}

// TestFileServiceModuleBoundaries drives every HTTP endpoint through the real
// application and checks the module boundary report.
func TestFileServiceModuleBoundaries(t *testing.T) {
	app, base := startApp(t)

	do(t, http.MethodGet, base+"/health", "", nil, http.StatusOK)

	// Single upload
	body, ct := multipartBody(t, "file", map[string]string{"hello.txt": "hello, mono"})
	var uploaded fileservice.UploadResult
	if err := json.Unmarshal(do(t, http.MethodPost, base+"/api/v1/files", ct, body, http.StatusCreated), &uploaded); err != nil {
		t.Fatal(err)
	}
	id := uploaded.FileInfo.ID
	if id == "" || uploaded.FileInfo.Name != "hello.txt" {
		t.Fatalf("unexpected upload result %+v", uploaded)
	}

	// Batch upload
	body, ct = multipartBody(t, "files", map[string]string{"a.json": "{}", "b.txt": "b"})
	var batch struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(do(t, http.MethodPost, base+"/api/v1/files/batch", ct, body, http.StatusCreated), &batch); err != nil || batch.Count != 2 {
		t.Fatalf("batch upload: %+v %v", batch, err)
	}

	// List, info, download
	var list fileservice.ListResult
	if err := json.Unmarshal(do(t, http.MethodGet, base+"/api/v1/files", "", nil, http.StatusOK), &list); err != nil || list.Total != 3 {
		t.Fatalf("list: %+v %v", list, err)
	}
	do(t, http.MethodGet, base+"/api/v1/files/"+id+"/info", "", nil, http.StatusOK)
	if got := string(do(t, http.MethodGet, base+"/api/v1/files/"+id, "", nil, http.StatusOK)); got != "hello, mono" {
		t.Errorf("downloaded %q", got)
	}

	// Delete, then the error mapping
	do(t, http.MethodDelete, base+"/api/v1/files/"+id, "", nil, http.StatusOK)
	do(t, http.MethodGet, base+"/api/v1/files/"+id, "", nil, http.StatusNotFound)
	do(t, http.MethodGet, base+"/api/v1/files/not-a-uuid/info", "", nil, http.StatusBadRequest)

	for _, r := range app.ModuleBoundaries() {
		if len(r.Violations) != 0 {
			t.Errorf("module %s boundary violations: %+v", r.Module, r.Violations)
		}
	}
}
