package taiga

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeleteMilestoneVerifiesNotFound(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/milestones/8":
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/milestones/8" && deleted:
			http.Error(w, `{"detail":"Not found"}`, http.StatusNotFound)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	if err := client.DeleteMilestone(context.Background(), 8); err != nil {
		t.Fatal(err)
	}
}

func TestCommentUndeleteAndVersions(t *testing.T) {
	undeleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/history/task/9/undelete_comment":
			if r.URL.Query().Get("id") != "h1" {
				t.Fatalf("query=%s", r.URL.RawQuery)
			}
			undeleted = true
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/history/task/9":
			date := "2026-09-01T00:00:00Z"
			if undeleted {
				date = ""
			}
			_, _ = io.WriteString(w, `[{"id":"h1","comment":"restored","delete_comment_date":"`+date+`"}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/history/task/9/comment_versions":
			if r.URL.Query().Get("id") != "h1" {
				t.Fatalf("query=%s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `[{"id":"v1","comment":"old","created_at":"2026-09-01T00:00:00Z"}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	entry, err := client.UndeleteComment(context.Background(), "task", 9, "h1")
	if err != nil || entry.DeleteCommentDate != "" {
		t.Fatalf("entry=%#v err=%v", entry, err)
	}
	versions, err := client.CommentVersions(context.Background(), "task", 9, "h1")
	if err != nil || len(versions) != 1 || versions[0].Comment != "old" {
		t.Fatalf("versions=%#v err=%v", versions, err)
	}
}

func TestCSVTokenDownloadAndRevoke(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/projects/2/regenerate_issues_csv_uuid":
			_, _ = io.WriteString(w, `{"uuid":"csv-token"}`)
		case "/api/v1/issues/csv":
			if r.URL.Query().Get("uuid") != "csv-token" || r.Header.Get("Authorization") != "Bearer secret" {
				t.Fatalf("request=%s auth=%q", r.URL.String(), r.Header.Get("Authorization"))
			}
			_, _ = io.WriteString(w, "ref,subject\n1,Demo\n")
		case "/api/v1/projects/2/delete_issues_csv_uuid":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/api/v1/", WithToken("secret"))
	token, err := client.CreateCSVToken(context.Background(), 2, "issue")
	if err != nil || token != "csv-token" {
		t.Fatalf("token=%q err=%v", token, err)
	}
	var output bytes.Buffer
	result, err := client.DownloadCSV(context.Background(), "issue", token, &output)
	if err != nil || result.Bytes != int64(output.Len()) || result.SHA256 == "" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if err := client.RevokeCSVToken(context.Background(), 2, "issue"); err != nil {
		t.Fatal(err)
	}
}

func TestProjectAdminResourceContracts(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/userstory-due-dates":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["project"] != float64(3) || body["days_to_due"] != float64(5) {
				t.Fatalf("body=%#v", body)
			}
			_, _ = io.WriteString(w, `{"id":4,"project":3,"name":"Soon","days_to_due":5}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/swimlane-userstory-statuses/7":
			_, _ = io.WriteString(w, `{"id":7,"swimlane":2,"status":9,"wip_limit":4}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/3/mix_tags":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["to_tag"] != "target" {
				t.Fatalf("body=%#v", body)
			}
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	due, err := client.CreateDueDate(context.Background(), "story", 3, map[string]any{"name": "Soon", "days_to_due": 5})
	if err != nil || due.ID != 4 {
		t.Fatalf("due=%#v err=%v", due, err)
	}
	limit := 4
	wip, err := client.UpdateSwimlaneWIP(context.Background(), 7, &limit)
	if err != nil || wip.WIPLimit == nil || *wip.WIPLimit != 4 {
		t.Fatalf("wip=%#v err=%v", wip, err)
	}
	if err := client.MixProjectTags(context.Background(), 3, []string{"a", "b"}, "target"); err != nil {
		t.Fatal(err)
	}
	if requests != 3 {
		t.Fatalf("requests=%d", requests)
	}
}

func TestAccountImporterAndBatchContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/user-storage/prefs":
			if r.Method != http.MethodPatch {
				t.Fatalf("method=%s", r.Method)
			}
			_, _ = io.WriteString(w, `{"key":"prefs","value":{"theme":"dark"}}`)
		case "/api/v1/importers/github/list_projects":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["token"] != "credential" {
				t.Fatalf("body=%#v", body)
			}
			_, _ = io.WriteString(w, `[{"id":1,"name":"repo"}]`)
		case "/api/v1/userstories/bulk_update_milestone":
			data, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(data), `"us_id":11`) || !strings.Contains(string(data), `"milestone_id":8`) {
				t.Fatalf("body=%s", data)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	entry, err := client.UpdateStorage(context.Background(), "prefs", map[string]any{"theme": "dark"})
	if err != nil || entry.Key != "prefs" {
		t.Fatalf("entry=%#v err=%v", entry, err)
	}
	result, err := client.ImporterCall(context.Background(), "github", "list-projects", map[string]any{"token": "credential"})
	if err != nil || result == nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if err := client.BulkMoveToMilestone(context.Background(), "story", 3, 8, []int64{11}); err != nil {
		t.Fatal(err)
	}
}
