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

func TestWorkItemDeleteRequiresConfirmationAndVerifies(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":1,"name":"Demo","slug":"demo"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issues/by_ref":
			_, _ = io.WriteString(w, `{"id":7,"ref":3,"project":1,"subject":"Delete me"}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/issues/7":
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issues/7" && deleted:
			http.Error(w, `{"detail":"Not found"}`, http.StatusNotFound)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	app, out, stderr, _ := testApp(t, server)
	code := app.Execute(context.Background(), []string{"--json", "--no-input", "issue", "delete", "3"})
	if code != ExitConfirmationRequired || out.Len() != 0 || deleted || !strings.Contains(stderr.String(), "confirmation_required") {
		t.Fatalf("exit=%d deleted=%t stdout=%s stderr=%s", code, deleted, out.String(), stderr.String())
	}
	app, out, stderr, _ = testApp(t, server)
	code = app.Execute(context.Background(), []string{"--json", "issue", "delete", "3", "--yes"})
	if code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data["deleted"] != true || envelope.Data["verified"] != true {
		t.Fatalf("data=%#v", envelope.Data)
	}
}

func TestParticipantCommandReturnsUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":1,"name":"Demo","slug":"demo"}`)
		case "/api/v1/issues/by_ref":
			_, _ = io.WriteString(w, `{"id":7,"ref":3,"project":1,"subject":"Issue"}`)
		case "/api/v1/issues/7/watchers":
			_, _ = io.WriteString(w, `[{"id":1,"username":"demo","full_name":"Demo User"}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	app, out, stderr, _ := testApp(t, server)
	if code := app.Execute(context.Background(), []string{"--json", "issue", "watchers", "3"}); code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	var envelope struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Items) != 1 || envelope.Items[0]["username"] != "demo" {
		t.Fatalf("items=%#v", envelope.Items)
	}
}

func TestCommentEditAndDeleteCommandsVerifyState(t *testing.T) {
	comment := "old"
	edited, deleted := false, false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":1,"name":"Demo","slug":"demo"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issues/by_ref":
			_, _ = io.WriteString(w, `{"id":7,"ref":3,"project":1,"subject":"Issue"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/history/issue/7":
			entry := map[string]any{"id": "entry-1", "comment": comment, "user": map[string]any{"pk": 1, "username": "demo"}}
			if edited {
				entry["edit_comment_date"] = "2026-09-01T00:00:00Z"
			}
			if deleted {
				entry["delete_comment_date"] = "2026-09-01T00:01:00Z"
			}
			_ = json.NewEncoder(w).Encode([]any{entry})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/history/issue/7/edit_comment":
			comment, edited = "new comment", true
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/history/issue/7/delete_comment":
			deleted = true
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	app, out, stderr, _ := testApp(t, server)
	if code := app.Execute(context.Background(), []string{"--json", "comment", "edit", "issue", "3", "entry-1", "--body", "new comment"}); code != ExitSuccess {
		t.Fatalf("edit exit=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(out.String(), `"verified":true`) || !strings.Contains(out.String(), "new comment") {
		t.Fatalf("edit output=%s", out.String())
	}
	app, out, stderr, _ = testApp(t, server)
	if code := app.Execute(context.Background(), []string{"--json", "comment", "delete", "issue", "3", "entry-1", "--yes"}); code != ExitSuccess {
		t.Fatalf("delete exit=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(out.String(), `"verified":true`) || !strings.Contains(out.String(), `"comment_deleted":true`) {
		t.Fatalf("delete output=%s", out.String())
	}
}
