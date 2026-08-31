package taiga

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseWikiRef(t *testing.T) {
	tests := []struct {
		name, value, project string
		want                 WikiRef
		wantErr              bool
	}{
		{name: "bare", value: "api-guide", project: "demo", want: WikiRef{Project: "demo", Slug: "api-guide"}},
		{name: "qualified", value: "other#runbook", want: WikiRef{Project: "other", Slug: "runbook"}},
		{name: "url", value: "https://example.test/taiga/project/demo/wiki/start-here", want: WikiRef{Project: "demo", Slug: "start-here"}},
		{name: "missing project", value: "start-here", wantErr: true},
		{name: "invalid slash", value: "folder/page", project: "demo", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseWikiRef(test.value, test.project)
			if (err != nil) != test.wantErr {
				t.Fatalf("ParseWikiRef() error=%v wantErr=%t", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("ParseWikiRef()=%#v want=%#v", got, test.want)
			}
		})
	}
}

func TestWikiClientCRUD(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/wiki":
			if r.URL.Query().Get("project") != "1" || r.URL.Query().Get("page") != "2" || r.URL.Query().Get("page_size") != "10" {
				t.Fatalf("list query=%s", r.URL.RawQuery)
			}
			w.Header().Set("X-Pagination-Current", "2")
			w.Header().Set("X-Pagination-Count", "11")
			_, _ = io.WriteString(w, `[{"id":7,"project":1,"slug":"guide","content":"hello","version":3}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/wiki/by_slug":
			if r.URL.Query().Get("project") != "1" || r.URL.Query().Get("slug") != "guide" {
				t.Fatalf("by_slug query=%s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"id":7,"project":1,"slug":"guide","content":"hello","version":3}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/wiki":
			var request CreateWikiPageRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Project != 1 || request.Slug != "new-page" || request.Content != "body" {
				t.Fatalf("create request=%#v", request)
			}
			_, _ = io.WriteString(w, `{"id":8,"project":1,"slug":"new-page","content":"body","version":1}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/wiki/7":
			var request UpdateWikiPageRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Version != 3 || request.Content == nil || *request.Content != "updated" {
				t.Fatalf("update request=%#v", request)
			}
			_, _ = io.WriteString(w, `{"id":7,"project":1,"slug":"guide","content":"updated","version":4}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/wiki/7":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	pages, page, err := client.ListWikiPages(context.Background(), 1, 2, 10)
	if err != nil || len(pages) != 1 || page.Number != 2 || page.Total != 11 {
		t.Fatalf("pages=%#v page=%#v err=%v", pages, page, err)
	}
	if _, err := client.GetWikiPageBySlug(context.Background(), 1, "guide"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateWikiPage(context.Background(), CreateWikiPageRequest{Project: 1, Slug: "new-page", Content: "body"}); err != nil {
		t.Fatal(err)
	}
	content := "updated"
	if _, err := client.UpdateWikiPage(context.Background(), 7, UpdateWikiPageRequest{Version: 3, Content: &content}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteWikiPage(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
}
