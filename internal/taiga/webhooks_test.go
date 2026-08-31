package taiga

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/webhooks":
			_, _ = io.WriteString(w, `[{"id":3,"project":1,"name":"CI","url":"https://example.test/hook","key":"secret","logs_counter":0}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/webhooks/3":
			_, _ = io.WriteString(w, `{"id":3,"project":1,"name":"CI","url":"https://example.test/hook","key":"secret"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/webhooks":
			var request CreateWebhookRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Project != 1 || request.Key != "secret" {
				t.Fatalf("create=%#v", request)
			}
			_, _ = io.WriteString(w, `{"id":3,"project":1,"name":"CI","url":"https://example.test/hook","key":"secret"}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/webhooks/3":
			_, _ = io.WriteString(w, `{"id":3,"project":1,"name":"CI 2","url":"https://example.test/hook","key":"secret"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/webhooks/3/test":
			_, _ = io.WriteString(w, `{"id":4,"webhook":3,"url":"https://example.test/hook","status":200,"duration":0.1}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/webhooks/3":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	if hooks, err := client.ListWebhooks(context.Background(), 1); err != nil || len(hooks) != 1 {
		t.Fatalf("hooks=%#v err=%v", hooks, err)
	}
	if _, err := client.GetWebhook(context.Background(), 3); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateWebhook(context.Background(), CreateWebhookRequest{Project: 1, Name: "CI", URL: "https://example.test/hook", Key: "secret"}); err != nil {
		t.Fatal(err)
	}
	name := "CI 2"
	if _, err := client.UpdateWebhook(context.Background(), 3, UpdateWebhookRequest{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if log, err := client.TestWebhook(context.Background(), 3); err != nil || log.Status != 200 {
		t.Fatalf("log=%#v err=%v", log, err)
	}
	if err := client.DeleteWebhook(context.Background(), 3); err != nil {
		t.Fatal(err)
	}
}
