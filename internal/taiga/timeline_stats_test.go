package taiga

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectTimelineUsesFiltersAndPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/timeline/project/7" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("only_relevant") != "true" || r.URL.Query().Get("page") != "2" || r.URL.Query().Get("page_size") != "10" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("X-Pagination-Current", "2")
		w.Header().Set("X-Paginated-By", "10")
		w.Header().Set("X-Pagination-Count", "11")
		_, _ = io.WriteString(w, `[{"id":9,"event_type":"issues.issue.change","project":7,"created":"2026-09-01T00:00:00Z","data":{"issue":{"id":2,"ref":3,"subject":"Issue"},"user":{"id":1,"username":"demo"},"values_diff":{"status":["New","Closed"]}}}]`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	entries, page, err := client.ProjectTimeline(context.Background(), 7, true, 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].EventType != "issues.issue.change" || page.Number != 2 || page.Total != 11 {
		t.Fatalf("entries=%#v page=%#v", entries, page)
	}
}

func TestStatsEndpointsDecodeStableContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/projects/7/stats":
			_, _ = io.WriteString(w, `{"name":"Demo","total_milestones":2,"total_points":20,"closed_points":8,"defined_points":20,"assigned_points":13,"speed":4,"milestones":[{"name":"Sprint 1","optimal":20,"evolution":20,"team-increment":0,"client-increment":0}]}`)
		case "/api/v1/projects/7/issues_stats":
			_, _ = io.WriteString(w, `{"total_issues":3,"opened_issues":2,"closed_issues":1,"issues_per_status":{"4":{"id":4,"name":"New","count":2}},"last_four_weeks_days":{"by_open_closed":{"open":[1],"closed":[0]},"by_severity":{},"by_priority":{},"by_status":{}}}`)
		case "/api/v1/projects/7/member_stats":
			_, _ = io.WriteString(w, `{"closed_bugs":{"1":2},"iocaine_tasks":{"1":0},"wiki_changes":{"1":3},"created_bugs":{"1":4},"closed_tasks":{"1":5}}`)
		case "/api/v1/milestones/8/stats":
			_, _ = io.WriteString(w, `{"name":"Sprint 1","estimated_start":"2026-09-01","estimated_finish":"2026-09-07","total_points":{"1":8},"completed_points":[3],"total_userstories":2,"completed_userstories":1,"total_tasks":4,"completed_tasks":2,"iocaine_doses":0,"days":[{"day":"2026-09-01","name":1,"open_points":8,"optimal_points":8}]}`)
		case "/api/v1/stats/system":
			_, _ = io.WriteString(w, `{"users":{"total":10},"projects":{"total":4,"total_with_backlog":2},"userstories":{"total":20}}`)
		case "/api/v1/stats/discover":
			_, _ = io.WriteString(w, `{"projects":{"total":3}}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	project, err := client.GetProjectStats(context.Background(), 7)
	if err != nil || project.DefinedPoints != 20 || project.Speed != 4 || len(project.Milestones) != 1 {
		t.Fatalf("project=%#v err=%v", project, err)
	}
	issues, err := client.GetProjectIssueStats(context.Background(), 7)
	if err != nil || issues.TotalIssues != 3 || issues.IssuesPerStatus["4"].Count != 2 {
		t.Fatalf("issues=%#v err=%v", issues, err)
	}
	members, err := client.GetProjectMemberStats(context.Background(), 7)
	if err != nil || members.ClosedTasks["1"] != 5 {
		t.Fatalf("members=%#v err=%v", members, err)
	}
	sprint, err := client.GetSprintStats(context.Background(), 8)
	if err != nil || sprint.TotalTasks != 4 || len(sprint.Days) != 1 {
		t.Fatalf("sprint=%#v err=%v", sprint, err)
	}
	system, err := client.GetSystemStats(context.Background())
	if err != nil || system.Users.Total != 10 || system.Projects.Total != 4 {
		t.Fatalf("system=%#v err=%v", system, err)
	}
	discover, err := client.GetDiscoverStats(context.Background())
	if err != nil || discover.Projects.Total != 3 {
		t.Fatalf("discover=%#v err=%v", discover, err)
	}
}
