package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTimelineCommandNormalizesEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":7,"name":"Demo","slug":"demo"}`)
		case "/api/v1/timeline/project/7":
			if r.URL.Query().Get("only_relevant") != "true" || r.URL.Query().Get("page_size") != "5" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			w.Header().Set("X-Pagination-Count", "1")
			_, _ = io.WriteString(w, `[{"id":9,"event_type":"issues.issue.change","project":7,"created":"2026-09-01T00:00:00Z","data":{"issue":{"id":2,"ref":3,"subject":"Issue"},"user":{"id":1,"name":"Demo User","username":"demo"},"comment":"done","values_diff":{"status":["New","Closed"]}}}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	app, out, stderr, _ := testApp(t, server)
	if code := app.Execute(context.Background(), []string{"--json", "timeline", "--limit", "5"}); code != ExitSuccess {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	var envelope struct {
		Items []timelineView `json:"items"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Items) != 1 || envelope.Items[0].Resource != "issue" || envelope.Items[0].Action != "change" || envelope.Items[0].Ref != 3 || envelope.Items[0].Username != "demo" || envelope.Items[0].Comment != "done" {
		t.Fatalf("items = %#v", envelope.Items)
	}
}

func TestStatsCommandContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/projects/by_slug":
			_, _ = io.WriteString(w, `{"id":7,"name":"Demo","slug":"demo"}`)
		case "/api/v1/projects/7/stats":
			_, _ = io.WriteString(w, `{"name":"Demo","defined_points":20,"assigned_points":13,"closed_points":8,"speed":4,"milestones":[]}`)
		case "/api/v1/projects/7/issues_stats":
			_, _ = io.WriteString(w, `{"total_issues":3,"opened_issues":2,"closed_issues":1,"last_four_weeks_days":{"by_open_closed":{"open":[],"closed":[]},"by_severity":{},"by_priority":{},"by_status":{}}}`)
		case "/api/v1/projects/7/member_stats":
			_, _ = io.WriteString(w, `{"closed_bugs":{"1":2},"iocaine_tasks":{"1":0},"wiki_changes":{"1":3},"created_bugs":{"1":4},"closed_tasks":{"1":5}}`)
		case "/api/v1/users":
			_, _ = io.WriteString(w, `[{"id":1,"username":"demo","full_name_display":"Demo User"}]`)
		case "/api/v1/milestones":
			_, _ = io.WriteString(w, `[{"id":8,"project":7,"name":"Sprint 1","slug":"sprint-1"}]`)
		case "/api/v1/milestones/8/stats":
			_, _ = io.WriteString(w, `{"name":"Sprint 1","total_points":{},"completed_points":[],"total_userstories":2,"completed_userstories":1,"total_tasks":4,"completed_tasks":2,"days":[]}`)
		case "/api/v1/stats/system":
			_, _ = io.WriteString(w, `{"users":{"total":10},"projects":{"total":4},"userstories":{"total":20}}`)
		case "/api/v1/stats/discover":
			_, _ = io.WriteString(w, `{"projects":{"total":3}}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	commands := [][]string{
		{"--json", "stats", "project"},
		{"--json", "stats", "issues", "demo"},
		{"--json", "stats", "members"},
		{"--json", "stats", "sprint", "sprint-1"},
		{"--json", "stats", "system"},
		{"--json", "stats", "discover"},
	}
	for _, args := range commands {
		app, out, stderr, _ := testApp(t, server)
		if code := app.Execute(context.Background(), args); code != ExitSuccess {
			t.Fatalf("taiga %v exit=%d stderr=%s", args, code, stderr.String())
		}
		if !json.Valid(out.Bytes()) {
			t.Fatalf("taiga %v returned invalid JSON: %s", args, out.String())
		}
	}
}
