package taiga

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEpicClientCRUDAndRelations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/epics":
			if r.URL.Query().Get("project") != "1" || r.URL.Query().Get("page") != "2" {
				t.Fatalf("list query=%s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `[{"id":7,"ref":8,"project":1,"subject":"Epic","version":1}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/epics/by_ref":
			_, _ = io.WriteString(w, `{"id":7,"ref":8,"project":1,"subject":"Epic","version":1}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/epics":
			var request CreateEpicRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Project != 1 || request.Subject != "New epic" {
				t.Fatalf("create=%#v", request)
			}
			_, _ = io.WriteString(w, `{"id":9,"ref":10,"project":1,"subject":"New epic","version":1}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/epics/7":
			var request UpdateEpicRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Version != 1 || request.Subject == nil || *request.Subject != "Updated" {
				t.Fatalf("update=%#v", request)
			}
			_, _ = io.WriteString(w, `{"id":7,"ref":8,"project":1,"subject":"Updated","version":2}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/epic-statuses":
			_, _ = io.WriteString(w, `[{"id":3,"name":"Closed","is_closed":true,"order":1}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/epics/7/related_userstories":
			_, _ = io.WriteString(w, `[{"epic":7,"user_story":11,"order":5}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/epics/7/related_userstories":
			var request CreateEpicRelatedUserStoryRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Epic != 7 || request.UserStory != 11 {
				t.Fatalf("link=%#v", request)
			}
			_, _ = io.WriteString(w, `{"epic":7,"user_story":11,"order":5}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/epics/7/related_userstories/11":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	if epics, _, err := client.ListEpics(context.Background(), 1, 2, 10); err != nil || len(epics) != 1 {
		t.Fatalf("epics=%#v err=%v", epics, err)
	}
	if _, err := client.GetEpicByRef(context.Background(), "demo", 8); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateEpic(context.Background(), CreateEpicRequest{Project: 1, Subject: "New epic"}); err != nil {
		t.Fatal(err)
	}
	subject := "Updated"
	if _, err := client.UpdateEpic(context.Background(), 7, UpdateEpicRequest{Version: 1, Subject: &subject}); err != nil {
		t.Fatal(err)
	}
	if statuses, err := client.EpicStatuses(context.Background(), 1); err != nil || len(statuses) != 1 || !statuses[0].IsClosed {
		t.Fatalf("statuses=%#v err=%v", statuses, err)
	}
	if related, err := client.ListEpicRelatedUserStories(context.Background(), 7); err != nil || len(related) != 1 {
		t.Fatalf("related=%#v err=%v", related, err)
	}
	if _, err := client.LinkEpicUserStory(context.Background(), 7, 11); err != nil {
		t.Fatal(err)
	}
	if err := client.UnlinkEpicUserStory(context.Background(), 7, 11); err != nil {
		t.Fatal(err)
	}
}

func TestParseEpicRefURL(t *testing.T) {
	got, err := ParseEpicRef("https://example.test/taiga/project/demo/epic/13", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != (ItemRef{Project: "demo", Ref: 13}) {
		t.Fatalf("ParseEpicRef()=%#v", got)
	}
}
