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

func TestBatchCreateDryRunResolvesMetadataWithoutWriting(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
		}
		switch r.URL.Path {
		case "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":7,"name":"Demo","slug":"demo"}`)
		case "/api/v1/issue-statuses":
			_, _ = io.WriteString(w, `[{"id":10,"name":"New","is_closed":false}]`)
		case "/api/v1/milestones":
			_, _ = io.WriteString(w, `[{"id":8,"project":7,"name":"Sprint 1","slug":"sprint-1"}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "issues.txt")
	if err := os.WriteFile(path, []byte("First issue\n\n Second issue \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app, out, stderr, _ := testApp(t, server)
	code := app.Execute(context.Background(), []string{"--json", "batch", "create", "issue", path, "--status", "New", "--sprint", "sprint-1", "--dry-run"})
	if code != ExitSuccess || posts != 0 {
		t.Fatalf("exit=%d posts=%d stderr=%s", code, posts, stderr.String())
	}
	var envelope struct {
		Plan struct {
			Changes map[string]any `json:"changes"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Plan.Changes["count"] != float64(2) || envelope.Plan.Changes["status"] != "New" || envelope.Plan.Changes["sprint"] != "sprint-1" {
		t.Fatalf("plan = %#v", envelope.Plan)
	}
}

func TestBatchCreateRequiresConfirmationBeforeReadingStdin(t *testing.T) {
	input := strings.NewReader("One\nTwo\n")
	app, out, stderr, _ := testApp(t, nil)
	app.In = input
	code := app.Execute(context.Background(), []string{"--json", "batch", "create", "story", "-"})
	if code != ExitConfirmationRequired || out.Len() != 0 || input.Len() != len("One\nTwo\n") || !strings.Contains(stderr.String(), "confirmation_required") {
		t.Fatalf("exit=%d stdout=%s stderr=%s unread=%d", code, out.String(), stderr.String(), input.Len())
	}
}

func TestBatchCreateReturnsVerifiedItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":7,"name":"Demo","slug":"demo"}`)
		case "/api/v1/userstories/bulk_create":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `[{"id":1,"ref":2,"project":7,"subject":"One","version":1,"status_extra_info":{"name":"New"}},{"id":2,"ref":3,"project":7,"subject":"Two","version":1,"status_extra_info":{"name":"New"}}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "stories.txt")
	if err := os.WriteFile(path, []byte("One\nTwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app, out, stderr, _ := testApp(t, server)
	code := app.Execute(context.Background(), []string{"--json", "batch", "create", "story", path, "--yes"})
	if code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	var envelope struct {
		Items []batchCreatedView `json:"items"`
		Page  map[string]any     `json:"page"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Items) != 2 || envelope.Items[0].Resource != "story" || envelope.Items[1].Ref != 3 || envelope.Page["verified"] != true {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestReadBatchSubjectsBoundsInput(t *testing.T) {
	if _, err := readBatchSubjects(strings.NewReader("\n\n"), "-"); err == nil {
		t.Fatal("expected empty batch error")
	}
	tooMany := strings.Repeat("item\n", maxBatchItems+1)
	if _, err := readBatchSubjects(strings.NewReader(tooMany), "-"); err == nil {
		t.Fatal("expected batch size error")
	}
}
