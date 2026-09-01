package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCSVExportWritesSecureFileAndRevokesToken(t *testing.T) {
	revoked := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":1,"name":"Demo","slug":"demo"}`)
		case "/api/v1/projects/1/regenerate_issues_csv_uuid":
			_, _ = io.WriteString(w, `{"uuid":"csv-token"}`)
		case "/api/v1/issues/csv":
			_, _ = io.WriteString(w, "ref,subject\n1,Demo\n")
		case "/api/v1/projects/1/delete_issues_csv_uuid":
			revoked = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	app, out, stderr, _ := testApp(t, server)
	path := filepath.Join(t.TempDir(), "issues.csv")
	if code := app.Execute(context.Background(), []string{"--json", "csv", "export", "issue", "--output", path}); code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ref,subject\n1,Demo\n" || !revoked || !strings.Contains(out.String(), `"token_revoked":true`) {
		t.Fatalf("data=%q revoked=%t out=%s", data, revoked, out.String())
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", stat.Mode().Perm())
	}
}

func TestSprintDeleteRequiresConfirmationAndVerifies(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":1,"name":"Demo","slug":"demo"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/milestones":
			_, _ = io.WriteString(w, `[{"id":5,"name":"Sprint 1","slug":"sprint-1","project":1}]`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/milestones/5":
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/milestones/5" && deleted:
			http.Error(w, `{"detail":"Not found"}`, http.StatusNotFound)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	app, out, stderr, _ := testApp(t, server)
	if code := app.Execute(context.Background(), []string{"--json", "sprint", "delete", "sprint-1", "--yes"}); code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	if !deleted || !strings.Contains(out.String(), `"verified":true`) {
		t.Fatalf("deleted=%t out=%s", deleted, out.String())
	}
}

func TestImporterDryRunRedactsCredentials(t *testing.T) {
	app, out, stderr, _ := testApp(t, nil)
	input := filepath.Join(t.TempDir(), "import.json")
	if err := os.WriteFile(input, []byte(`{"token":"top-secret","project":"42","name":"Imported"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := app.Execute(context.Background(), []string{"--json", "integration", "call", "github", "import-project", "--input", input, "--dry-run"}); code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	if strings.Contains(out.String(), "top-secret") || !strings.Contains(out.String(), "[REDACTED]") {
		t.Fatalf("output=%s", out.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
}
