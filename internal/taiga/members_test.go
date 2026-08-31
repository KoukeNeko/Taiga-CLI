package taiga

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMembershipAndRoleClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/memberships":
			_, _ = io.WriteString(w, `[{"id":3,"project":1,"role":4,"role_name":"Developer","user_email":"alice@example.com"}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/memberships":
			var request CreateMembershipRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Username != "alice@example.com" || request.Project != 1 || request.Role != 4 {
				t.Fatalf("membership create=%#v", request)
			}
			_, _ = io.WriteString(w, `{"id":3,"project":1,"role":4,"role_name":"Developer","user_email":"alice@example.com"}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/memberships/3":
			_, _ = io.WriteString(w, `{"id":3,"project":1,"role":4,"is_admin":true}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/memberships/3":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/roles":
			_, _ = io.WriteString(w, `[{"id":4,"project":1,"name":"Developer","slug":"developer","computable":true,"members_count":1}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/roles":
			_, _ = io.WriteString(w, `{"id":5,"project":1,"name":"Reviewer","slug":"reviewer","computable":false}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/roles/5":
			_, _ = io.WriteString(w, `{"id":5,"project":1,"name":"Review","slug":"reviewer","computable":false}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/roles/5":
			if r.URL.Query().Get("moveTo") != "4" {
				t.Fatalf("moveTo query=%s", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	if members, err := client.ListMemberships(context.Background(), 1); err != nil || len(members) != 1 {
		t.Fatalf("members=%#v err=%v", members, err)
	}
	if _, err := client.CreateMembership(context.Background(), CreateMembershipRequest{Username: "alice@example.com", Project: 1, Role: 4}); err != nil {
		t.Fatal(err)
	}
	admin := true
	if _, err := client.UpdateMembership(context.Background(), 3, UpdateMembershipRequest{IsAdmin: &admin}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteMembership(context.Background(), 3); err != nil {
		t.Fatal(err)
	}
	if roles, err := client.ListRoles(context.Background(), 1); err != nil || len(roles) != 1 {
		t.Fatalf("roles=%#v err=%v", roles, err)
	}
	if _, err := client.CreateRole(context.Background(), CreateRoleRequest{Name: "Reviewer", Project: 1}); err != nil {
		t.Fatal(err)
	}
	name := "Review"
	if _, err := client.UpdateRole(context.Background(), 5, UpdateRoleRequest{Name: &name}); err != nil {
		t.Fatal(err)
	}
	moveTo := int64(4)
	if err := client.DeleteRole(context.Background(), 5, &moveTo); err != nil {
		t.Fatal(err)
	}
}
