package taiga

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWorkflowMetadataCRUDAndMoveDelete(t *testing.T) {
	current := WorkflowMetadata{ID: 3, Project: 1, Name: "Review", Slug: "review", Order: 10, Color: "#112233"}
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issue-statuses":
			if r.URL.Query().Get("project") != "1" {
				t.Fatalf("query=%s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode([]WorkflowMetadata{current})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/issue-statuses":
			_ = json.NewEncoder(w).Encode(current)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/issue-statuses/3":
			current.Name = "QA"
			_ = json.NewEncoder(w).Encode(current)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/issue-statuses/3":
			if r.URL.Query().Get("moveTo") != "4" {
				t.Fatalf("delete query=%s", r.URL.RawQuery)
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
	client, _ := NewClient(server.URL + "/api/v1/")
	values, err := client.ListWorkflowMetadata(context.Background(), "issue-status", 1)
	if err != nil || len(values) != 1 {
		t.Fatalf("values=%#v err=%v", values, err)
	}
	if _, err := client.CreateWorkflowMetadata(context.Background(), "issue-status", 1, map[string]any{"name": "Review"}); err != nil {
		t.Fatal(err)
	}
	updated, err := client.UpdateWorkflowMetadata(context.Background(), "issue-status", 3, map[string]any{"name": "QA"})
	if err != nil || updated.Name != "QA" {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
	if err := client.DeleteWorkflowMetadata(context.Background(), "issue-status", 3, 4); err != nil {
		t.Fatal(err)
	}
}

func TestMetadataKindPaths(t *testing.T) {
	want := map[string]string{"epic-status": "epic-statuses", "story-status": "userstory-statuses", "task-status": "task-statuses", "issue-status": "issue-statuses", "points": "points", "priority": "priorities", "severity": "severities", "issue-type": "issue-types"}
	for kind, path := range want {
		got, err := MetadataPath(kind)
		if err != nil || got != path {
			t.Fatalf("kind=%s path=%s err=%v", kind, got, err)
		}
	}
	if _, err := NormalizeMetadataKind("unknown"); err == nil {
		t.Fatal("expected unknown metadata kind error")
	}
}
