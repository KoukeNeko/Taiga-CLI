package taiga

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteWorkItemVerifiesNotFound(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
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
	client, _ := NewClient(server.URL + "/api/v1/")
	if err := client.DeleteWorkItem(context.Background(), "issue", 7); err != nil {
		t.Fatal(err)
	}
}

func TestListParticipantsUsesResourceEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/userstories/7/voters" || r.URL.Query().Get("page") != "2" || r.URL.Query().Get("page_size") != "10" {
			t.Fatalf("request = %s", r.URL.String())
		}
		w.Header().Set("X-Pagination-Current", "2")
		w.Header().Set("X-Pagination-Count", "11")
		_, _ = io.WriteString(w, `[{"id":1,"username":"demo","full_name":"Demo User"}]`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	participants, page, err := client.ListParticipants(context.Background(), "story", 7, "voters", 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(participants) != 1 || participants[0].Username != "demo" || page.Number != 2 || page.Total != 11 {
		t.Fatalf("participants=%#v page=%#v", participants, page)
	}
}

func TestEditAndDeleteCommentVerifyHistory(t *testing.T) {
	comment := "old"
	edited, deleted := false, false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/history/issue/7/edit_comment":
			if r.URL.Query().Get("id") != "entry-1" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			comment, edited = body["comment"], true
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/history/issue/7/delete_comment":
			deleted = true
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/history/issue/7":
			entry := map[string]any{"id": "entry-1", "comment": comment, "edit_comment_date": "", "delete_comment_date": ""}
			if edited {
				entry["edit_comment_date"] = "2026-09-01T00:00:00Z"
			}
			if deleted {
				entry["delete_comment_date"] = "2026-09-01T00:01:00Z"
			}
			_ = json.NewEncoder(w).Encode([]any{entry})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	entry, err := client.EditComment(context.Background(), "issue", 7, "entry-1", "new")
	if err != nil || entry.Comment != "new" || entry.EditCommentDate == "" {
		t.Fatalf("entry=%#v err=%v", entry, err)
	}
	entry, err = client.DeleteComment(context.Background(), "issue", 7, "entry-1")
	if err != nil || entry.DeleteCommentDate == "" {
		t.Fatalf("entry=%#v err=%v", entry, err)
	}
}
