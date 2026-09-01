package taiga

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestProjectExportTransportFailureIsAmbiguous(t *testing.T) {
	client, _ := NewClient("https://example.test/api/v1/", WithHTTPClient(&http.Client{Transport: failingTransport{}}))
	_, err := client.RequestProjectExport(context.Background(), 7, "gzip")
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Kind != KindAmbiguousCommit {
		t.Fatalf("error = %#v", err)
	}
}

func TestRequestProjectExportUsesOneShotGET(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/exporter/7" || r.URL.Query().Get("dump_format") != "gzip" {
			t.Fatalf("request = %s %s", r.Method, r.URL.String())
		}
		http.Error(w, `{"detail":"temporary"}`, http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/api/v1/", WithMaxRetries(3))
	if _, err := client.RequestProjectExport(context.Background(), 7, "gzip"); err == nil {
		t.Fatal("expected export request error")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want one non-idempotent GET", requests)
	}
}

func TestRequestProjectExportParsesAcceptedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"export_id":"export-123"}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	result, err := client.RequestProjectExport(context.Background(), 7, "plain")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Accepted() || result.ExportID != "export-123" {
		t.Fatalf("result = %#v", result)
	}
}

func TestImportProjectDumpStreamsMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/importer/load_dump" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		file, header, err := r.FormFile("dump")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = file.Close() }()
		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatal(err)
		}
		if header.Filename != "project.json.gz" || header.Header.Get("Content-Type") != "application/gzip" || string(data) != "dump-data" {
			t.Fatalf("filename=%q content-type=%q data=%q", header.Filename, header.Header.Get("Content-Type"), data)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"import_id":"import-123"}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/api/v1/", WithToken("token"))
	result, err := client.ImportProjectDump(context.Background(), "project.json.gz", "application/gzip", strings.NewReader("dump-data"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Accepted() || result.ImportID != "import-123" {
		t.Fatalf("result = %#v", result)
	}
}
