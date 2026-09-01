package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetadataCommandLifecycle(t *testing.T) {
	values := []map[string]any{{"id": float64(3), "project": float64(1), "name": "Review", "slug": "review", "order": float64(10), "color": "#112233", "is_closed": false}, {"id": float64(4), "project": float64(1), "name": "Done", "slug": "done", "order": float64(20), "color": "#334455", "is_closed": true}}
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":1,"name":"Demo","slug":"demo"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issue-statuses":
			_ = json.NewEncoder(w).Encode(values)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/issue-statuses":
			_ = json.NewEncoder(w).Encode(values[0])
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/issue-statuses/3":
			values[0]["name"] = "QA"
			_ = json.NewEncoder(w).Encode(values[0])
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/issue-statuses/3":
			if r.URL.Query().Get("moveTo") != "4" {
				t.Fatalf("query=%s", r.URL.RawQuery)
			}
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issue-statuses/3" && deleted:
			http.Error(w, `{"detail":"Not found"}`, http.StatusNotFound)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	commands := [][]string{
		{"--json", "metadata", "list", "issue-status"},
		{"--json", "metadata", "view", "issue-status", "review"},
		{"--json", "metadata", "create", "issue-status", "--name", "Review", "--color", "#112233"},
		{"--json", "metadata", "edit", "issue-status", "review", "--name", "QA"},
		{"--json", "metadata", "delete", "issue-status", "QA", "--move-to", "Done", "--yes"},
	}
	for _, args := range commands {
		app, out, stderr, _ := testApp(t, server)
		if code := app.Execute(context.Background(), args); code != ExitSuccess {
			t.Fatalf("taiga %v exit=%d stderr=%s", args, code, stderr.String())
		}
		if !json.Valid(out.Bytes()) {
			t.Fatalf("taiga %v output=%s", args, out.String())
		}
	}
}

func TestMetadataRejectsInvalidKindFlags(t *testing.T) {
	app, out, stderr, _ := testApp(t, nil)
	code := app.Execute(context.Background(), []string{"--json", "metadata", "create", "points", "--name", "Three", "--color", "#112233"})
	if code != ExitUsage || out.Len() != 0 || !strings.Contains(stderr.String(), "not valid for Points") {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
	app, out, stderr, _ = testApp(t, nil)
	code = app.Execute(context.Background(), []string{"--json", "metadata", "delete", "priority", "High"})
	if code != ExitUsage || out.Len() != 0 || !strings.Contains(stderr.String(), "--move-to is required") {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
}
