package taiga

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCustomFieldClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issue-custom-attributes":
			_, _ = io.WriteString(w, `[{"id":3,"project":1,"name":"Environment","type":"text","order":1}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/issue-custom-attributes":
			var request CreateCustomFieldRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Project != 1 || request.Name != "Environment" {
				t.Fatalf("create=%#v", request)
			}
			_, _ = io.WriteString(w, `{"id":3,"project":1,"name":"Environment","type":"text"}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/issue-custom-attributes/3":
			_, _ = io.WriteString(w, `{"id":3,"project":1,"name":"Env","type":"text"}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/issue-custom-attributes/3":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issues/custom-attributes-values/7":
			_, _ = io.WriteString(w, `{"issue":7,"attributes_values":{"3":"staging"},"version":2}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/issues/custom-attributes-values/7":
			var request UpdateCustomFieldValuesRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Version != 2 || request.AttributesValues["3"] != "production" {
				t.Fatalf("values update=%#v", request)
			}
			_, _ = io.WriteString(w, `{"issue":7,"attributes_values":{"3":"production"},"version":3}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	if fields, err := client.ListCustomFields(context.Background(), "issue", 1); err != nil || len(fields) != 1 {
		t.Fatalf("fields=%#v err=%v", fields, err)
	}
	if _, err := client.CreateCustomField(context.Background(), "issue", CreateCustomFieldRequest{Project: 1, Name: "Environment", Type: "text"}); err != nil {
		t.Fatal(err)
	}
	name := "Env"
	if _, err := client.UpdateCustomField(context.Background(), "issue", 3, UpdateCustomFieldRequest{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteCustomField(context.Background(), "issue", 3); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetCustomFieldValues(context.Background(), "issue", 7); err != nil {
		t.Fatal(err)
	}
	if _, err := client.UpdateCustomFieldValues(context.Background(), "issue", 7, UpdateCustomFieldValuesRequest{AttributesValues: map[string]any{"3": "production"}, Version: 2}); err != nil {
		t.Fatal(err)
	}
}
