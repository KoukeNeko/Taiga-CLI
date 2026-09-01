package taiga

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBulkCreateResourceContracts(t *testing.T) {
	milestoneID, storyID, statusID := int64(8), int64(9), int64(10)
	tests := []struct {
		resource string
		path     string
		request  BulkCreateRequest
		bulkKey  string
		check    func(*testing.T, map[string]any)
	}{
		{resource: "epic", path: "/api/v1/epics/bulk_create", request: BulkCreateRequest{ProjectID: 7, Subjects: "One\nTwo", StatusID: &statusID}, bulkKey: "bulk_epics"},
		{resource: "story", path: "/api/v1/userstories/bulk_create", request: BulkCreateRequest{ProjectID: 7, Subjects: "One\nTwo"}, bulkKey: "bulk_stories"},
		{resource: "issue", path: "/api/v1/issues/bulk_create", request: BulkCreateRequest{ProjectID: 7, Subjects: "One\nTwo", MilestoneID: &milestoneID}, bulkKey: "bulk_issues", check: func(t *testing.T, body map[string]any) {
			if body["milestone_id"] != float64(8) {
				t.Fatalf("milestone_id = %#v", body["milestone_id"])
			}
		}},
		{resource: "task", path: "/api/v1/tasks/bulk_create", request: BulkCreateRequest{ProjectID: 7, Subjects: "One\nTwo", MilestoneID: &milestoneID, StoryID: &storyID}, bulkKey: "bulk_tasks", check: func(t *testing.T, body map[string]any) {
			if body["milestone_id"] != float64(8) || body["us_id"] != float64(9) {
				t.Fatalf("body = %#v", body)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.resource, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != test.path {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["project_id"] != float64(7) || body[test.bulkKey] != "One\nTwo" {
					t.Fatalf("body = %#v", body)
				}
				if test.check != nil {
					test.check(t, body)
				}
				_, _ = io.WriteString(w, `[{"id":1,"ref":2,"project":7,"subject":"One","version":1,"status_extra_info":{"name":"New"}},{"id":2,"ref":3,"project":7,"subject":"Two","version":1,"status_extra_info":{"name":"New"}}]`)
			}))
			defer server.Close()
			client, _ := NewClient(server.URL + "/api/v1/")
			created, err := client.BulkCreate(context.Background(), test.resource, test.request)
			if err != nil {
				t.Fatal(err)
			}
			if len(created) != 2 || created[0].Subject != "One" || created[1].Ref != 3 {
				t.Fatalf("created = %#v", created)
			}
		})
	}
}

func TestBulkCreateTaskRequiresMilestone(t *testing.T) {
	client, _ := NewClient("https://example.test/api/v1/")
	if _, err := client.BulkCreate(context.Background(), "task", BulkCreateRequest{ProjectID: 7, Subjects: "One"}); err == nil {
		t.Fatal("expected missing milestone error")
	}
}
