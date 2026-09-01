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

func TestApplicationListUsesAuthorizedTokensAndRedactsSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/application-tokens" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, `[{"id":7,"auth_code":"must-not-leak","next_url":"https://app.invalid/callback?auth_code=must-not-leak","application":{"id":"demo-app","name":"Demo App","web":"https://app.invalid"}}]`)
	}))
	defer server.Close()
	app, out, stderr, _ := testApp(t, server)
	if code := app.Execute(context.Background(), []string{"--json", "application", "list"}); code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	if strings.Contains(out.String(), "must-not-leak") || !strings.Contains(out.String(), `"id":"demo-app"`) {
		t.Fatalf("output=%s", out.String())
	}
}

func TestStorageKeyRejectsRouterReservedCharacters(t *testing.T) {
	app, out, stderr, _ := testApp(t, nil)
	if code := app.Execute(context.Background(), []string{"--json", "storage", "get", "dashboard.preferences"}); code != ExitUsage {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "cannot contain") {
		t.Fatalf("stderr=%s", stderr.String())
	}
}

func TestNotificationPolicyCreateUsesProvisionedPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":1,"name":"Demo","slug":"demo"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/notify-policies":
			_, _ = io.WriteString(w, `[{"id":4,"project":1,"project_name":"Demo","notify_level":2,"live_notify_level":1,"web_notify_level":true}]`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/notify-policies/4":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["notify_level"] != float64(1) || body["live_notify_level"] != float64(2) || body["web_notify_level"] != false {
				t.Fatalf("body=%#v", body)
			}
			_, _ = io.WriteString(w, `{"id":4,"project":1,"notify_level":1,"live_notify_level":2,"web_notify_level":false}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	app, out, stderr, _ := testApp(t, server)
	if code := app.Execute(context.Background(), []string{"--json", "notification", "policy", "create", "--email", "involved", "--live", "all", "--web=false"}); code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(out.String(), `"notify_level":1`) {
		t.Fatalf("output=%s", out.String())
	}
}

func TestCustomFieldUnsetLastValueUsesNull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":1,"name":"Demo","slug":"demo"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issues/by_ref":
			_, _ = io.WriteString(w, `{"id":7,"ref":3,"project":1,"subject":"Issue"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issue-custom-attributes":
			_, _ = io.WriteString(w, `[{"id":3,"project":1,"name":"Environment","type":"text"}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issues/custom-attributes-values/7":
			_, _ = io.WriteString(w, `{"version":2,"attributes_values":{"3":"production"}}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/issues/custom-attributes-values/7":
			var body struct {
				Version int            `json:"version"`
				Values  map[string]any `json:"attributes_values"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			value, exists := body.Values["3"]
			if body.Version != 2 || !exists || value != nil {
				t.Fatalf("body=%#v", body)
			}
			_, _ = io.WriteString(w, `{"version":3,"attributes_values":{"3":null}}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	app, out, stderr, _ := testApp(t, server)
	if code := app.Execute(context.Background(), []string{"--json", "custom-field", "set", "issue", "3", "--unset", "Environment"}); code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(out.String(), `"Environment":null`) {
		t.Fatalf("output=%s", out.String())
	}
}
