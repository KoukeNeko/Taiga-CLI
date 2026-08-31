package taiga

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectClientWriteOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/project-templates":
			_, _ = io.WriteString(w, `[{"id":2,"name":"Scrum","slug":"scrum","description":"Template"}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects":
			var request CreateProjectRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Name != "Demo" || request.CreationTemplate != 2 || !request.IsPrivate {
				t.Fatalf("create=%#v", request)
			}
			_, _ = io.WriteString(w, `{"id":7,"name":"Demo","slug":"demo","is_private":true}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/projects/7":
			var request UpdateProjectRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Description == nil || *request.Description != "Updated" || request.IsWikiActivated == nil || *request.IsWikiActivated {
				t.Fatalf("update=%#v", request)
			}
			_, _ = io.WriteString(w, `{"id":7,"name":"Demo","slug":"demo","description":"Updated","is_wiki_activated":false}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/projects/7":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	if templates, err := client.ListProjectTemplates(context.Background()); err != nil || len(templates) != 1 || templates[0].Slug != "scrum" {
		t.Fatalf("templates=%#v err=%v", templates, err)
	}
	if _, err := client.CreateProject(context.Background(), CreateProjectRequest{Name: "Demo", CreationTemplate: 2, IsPrivate: true}); err != nil {
		t.Fatal(err)
	}
	description, wiki := "Updated", false
	if _, err := client.UpdateProject(context.Background(), 7, UpdateProjectRequest{Description: &description, IsWikiActivated: &wiki}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteProject(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
}
