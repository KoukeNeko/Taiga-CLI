package taiga

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientSendsBearerAndParsesPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("Authorization = %q", got)
		}
		w.Header().Set("X-Pagination-Current", "2")
		w.Header().Set("X-Paginated-By", "10")
		w.Header().Set("X-Pagination-Count", "25")
		w.Header().Set("X-Pagination-Next", "3")
		_, _ = io.WriteString(w, `[{"id":1,"name":"Demo","slug":"demo"}]`)
	}))
	defer server.Close()
	client, err := NewClient(server.URL+"/api/v1/", WithToken("secret-token"))
	if err != nil {
		t.Fatal(err)
	}
	projects, page, err := client.ListProjects(context.Background(), 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].Slug != "demo" {
		t.Fatalf("projects = %#v", projects)
	}
	if page.Number != 2 || page.Size != 10 || page.Total != 25 || page.Next != 3 {
		t.Fatalf("page = %#v", page)
	}
}

func TestClientRetriesGETButNotMutation(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests < 3 {
			http.Error(w, `{"detail":"temporary"}`, http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/", WithMaxRetries(3))
	client.sleep = func(context.Context, time.Duration) error { return nil }
	var result []any
	if _, err := client.Get(context.Background(), "projects", nil, &result); err != nil {
		t.Fatal(err)
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection lost")
}

func TestMutationTransportFailureIsAmbiguous(t *testing.T) {
	httpClient := &http.Client{Transport: failingTransport{}}
	client, _ := NewClient("https://example.test/api/v1/", WithHTTPClient(httpClient))
	_, err := client.Post(context.Background(), "issues", map[string]any{"subject": "x"}, nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Kind != KindAmbiguousCommit {
		t.Fatalf("error = %#v", err)
	}
}

func TestAPIErrorClassificationAndNoTokenInVerbose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"_error_message":"wrong version","_error_type":"WrongVersion"}`)
	}))
	defer server.Close()
	var log strings.Builder
	client, _ := NewClient(server.URL+"/", WithToken("do-not-log"), WithVerbose(&log))
	_, err := client.Patch(context.Background(), "issues/1", UpdateIssueRequest{Version: 1}, nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Kind != KindConflict {
		t.Fatalf("error = %#v", err)
	}
	if strings.Contains(log.String(), "do-not-log") {
		t.Fatalf("verbose log leaked token: %s", log.String())
	}
}

func TestDiscoverAPIPreservesSubpath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/taiga/conf.json" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"api":"`+"http://"+r.Host+`/taiga/api/v1/","baseHref":"/taiga/"}`)
	}))
	defer server.Close()
	config, err := DiscoverAPI(context.Background(), server.Client(), server.URL+"/taiga/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(config.API, "/taiga/api/v1/") {
		t.Fatalf("API = %q", config.API)
	}
}

func TestClientRefreshesExpiredTokenAndPersistsRotation(t *testing.T) {
	meRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/users/me":
			meRequests++
			if r.Header.Get("Authorization") == "Bearer old-token" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"detail":"expired"}`)
				return
			}
			if r.Header.Get("Authorization") != "Bearer new-token" {
				t.Fatalf("unexpected refreshed Authorization: %q", r.Header.Get("Authorization"))
			}
			_, _ = io.WriteString(w, `{"id":1,"username":"demo"}`)
		case "/api/v1/auth/refresh":
			if r.Header.Get("Authorization") != "" {
				t.Fatalf("refresh leaked Authorization header")
			}
			_, _ = io.WriteString(w, `{"auth_token":"new-token","refresh":"new-refresh"}`)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	var persistedAuth, persistedRefresh string
	client, _ := NewClient(server.URL+"/api/v1/",
		WithToken("old-token"),
		WithRefreshToken("old-refresh", func(authToken, refreshToken string) error {
			persistedAuth, persistedRefresh = authToken, refreshToken
			return nil
		}),
	)
	user, err := client.Me(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "demo" || meRequests != 2 {
		t.Fatalf("user=%#v meRequests=%d", user, meRequests)
	}
	if persistedAuth != "new-token" || persistedRefresh != "new-refresh" {
		t.Fatalf("persisted auth=%q refresh=%q", persistedAuth, persistedRefresh)
	}
}

func TestClientPropagatesCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/", WithMaxRetries(3))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var output any
	_, err := client.Get(ctx, "slow", nil, &output)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %#v", err)
	}
}

func TestAPIStatusClassification(t *testing.T) {
	tests := []struct {
		status int
		kind   ErrorKind
	}{
		{status: http.StatusUnauthorized, kind: KindAuth},
		{status: http.StatusForbidden, kind: KindForbidden},
		{status: http.StatusNotFound, kind: KindNotFound},
		{status: http.StatusTooManyRequests, kind: KindThrottled},
		{status: http.StatusBadGateway, kind: KindTransport},
	}
	for _, test := range tests {
		t.Run(http.StatusText(test.status), func(t *testing.T) {
			apiErr := decodeAPIError("GET /resource", test.status, []byte(`{"detail":"failure"}`))
			if apiErr.Kind != test.kind {
				t.Fatalf("kind = %q, want %q", apiErr.Kind, test.kind)
			}
		})
	}
}

func TestTaigaVersionFieldIsConflict(t *testing.T) {
	apiErr := decodeAPIError("PATCH /issues/1", http.StatusBadRequest, []byte(`{"version":"The version doesn't match with the current one"}`))
	if apiErr.Kind != KindConflict {
		t.Fatalf("kind = %q, want %q", apiErr.Kind, KindConflict)
	}
}

func TestLoginUsesCurrentRefreshField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/auth" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var request LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Type != "normal" || request.Username != "demo" || request.Password != "password" {
			t.Fatalf("request = %#v", request)
		}
		_, _ = io.WriteString(w, `{"auth_token":"auth","refresh":"refresh","id":1,"username":"demo"}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	response, err := client.Login(context.Background(), "demo", "password")
	if err != nil {
		t.Fatal(err)
	}
	if response.AuthToken != "auth" || response.RefreshToken != "refresh" {
		t.Fatalf("response = %#v", response)
	}
}

func TestCreateAttachmentStreamsMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/issues/attachments" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("project") != "1" || r.FormValue("object_id") != "2" || r.FormValue("description") != "evidence" {
			t.Fatalf("form = %#v", r.MultipartForm.Value)
		}
		file, header, err := r.FormFile("attached_file")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = file.Close() }()
		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatal(err)
		}
		if header.Filename != "note.txt" || string(data) != "hello" {
			t.Fatalf("filename=%q data=%q", header.Filename, data)
		}
		_, _ = io.WriteString(w, `{"id":3,"project":1,"object_id":2,"name":"note.txt","size":5,"description":"evidence"}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/api/v1/", WithToken("token"))
	attachment, err := client.CreateAttachment(context.Background(), "issue", 1, 2, "note.txt", "evidence", false, strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if attachment.ID != 3 || attachment.Name != "note.txt" || attachment.Size != 5 {
		t.Fatalf("attachment = %#v", attachment)
	}
}

func TestWatchAndHistoryEndpoints(t *testing.T) {
	watching := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/issues/7/watch":
			watching = true
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/issues/7":
			_, _ = io.WriteString(w, fmt.Sprintf(`{"is_watcher":%t}`, watching))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/history/issue/7":
			if r.URL.Query().Get("type") != "comment" || r.URL.Query().Get("page") != "2" || r.URL.Query().Get("page_size") != "10" {
				t.Fatalf("history query = %s", r.URL.RawQuery)
			}
			w.Header().Set("X-Pagination-Current", "2")
			w.Header().Set("X-Paginated-By", "10")
			w.Header().Set("X-Pagination-Count", "11")
			_, _ = io.WriteString(w, `[{"id":"entry-1","type":1,"created_at":"2026-08-31T00:00:00Z","user":{"pk":1,"username":"demo","name":"Demo"},"comment":"note"}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	verified, err := client.SetWatching(context.Background(), "issue", 7, true)
	if err != nil || !verified {
		t.Fatalf("verified=%t err=%v", verified, err)
	}
	entries, page, err := client.History(context.Background(), "issue", 7, "comment", 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != "entry-1" || entries[0].User.Username != "demo" || page.Number != 2 || page.Total != 11 {
		t.Fatalf("entries=%#v page=%#v", entries, page)
	}
}

func TestVotingEndpoints(t *testing.T) {
	voting := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/epics/7/upvote":
			voting = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/epics/7/downvote":
			voting = false
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/epics/7":
			_, _ = fmt.Fprintf(w, `{"is_voter":%t}`, voting)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	for _, expected := range []bool{true, false} {
		verified, err := client.SetVoting(context.Background(), "epic", 7, expected)
		if err != nil || verified != expected {
			t.Fatalf("expected=%t verified=%t err=%v", expected, verified, err)
		}
	}
}

func TestAttachmentResourcePaths(t *testing.T) {
	tests := []struct {
		resource string
		path     string
	}{
		{resource: "epic", path: "/api/v1/epics/attachments"},
		{resource: "wiki", path: "/api/v1/wiki/attachments"},
	}
	for _, test := range tests {
		t.Run(test.resource, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != test.path {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				if r.URL.Query().Get("project") != "1" || r.URL.Query().Get("object_id") != "2" {
					t.Fatalf("query = %s", r.URL.RawQuery)
				}
				_, _ = io.WriteString(w, `[{"id":3,"project":1,"object_id":2,"name":"note.txt"}]`)
			}))
			defer server.Close()
			client, _ := NewClient(server.URL + "/api/v1/")
			attachments, err := client.ListAttachments(context.Background(), test.resource, 1, 2)
			if err != nil {
				t.Fatal(err)
			}
			if len(attachments) != 1 || attachments[0].ID != 3 {
				t.Fatalf("attachments = %#v", attachments)
			}
		})
	}
}

func TestRetryDelayHonoursRetryAfter(t *testing.T) {
	if got := retryDelay(1, "5"); got != 5*time.Second {
		t.Fatalf("retryDelay(1, \"5\") = %v, want 5s", got)
	}
	if got := retryDelay(1, "120"); got != maxRetryAfter {
		t.Fatalf("retryDelay(1, \"120\") = %v, want %v", got, maxRetryAfter)
	}
	past := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	if got := retryDelay(1, past); got != 0 {
		t.Fatalf("retryDelay(1, past) = %v, want 0", got)
	}
	future := time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)
	if got := retryDelay(1, future); got != maxRetryAfter {
		t.Fatalf("retryDelay(1, future) = %v, want %v", got, maxRetryAfter)
	}
}

func TestRetryDelayBackoffIsBoundedAndJittered(t *testing.T) {
	const samples = 200
	for attempt := 1; attempt <= 4; attempt++ {
		base := backoffBase(attempt)
		distinct := map[time.Duration]struct{}{}
		for sample := 0; sample < samples; sample++ {
			delay := retryDelay(attempt, "")
			if delay < base || delay >= base+jitterWindow {
				t.Fatalf("retryDelay(%d, \"\") = %v, want [%v, %v)", attempt, delay, base, base+jitterWindow)
			}
			distinct[delay] = struct{}{}
		}
		if len(distinct) < 2 {
			t.Fatalf("attempt %d produced a single delay across %d samples; jitter is not random", attempt, samples)
		}
	}
}

func TestBackoffBaseDoublesWithoutOverflowing(t *testing.T) {
	if backoffBase(1) != backoffStep || backoffBase(3) != 4*backoffStep {
		t.Fatalf("backoffBase(1) = %v, backoffBase(3) = %v", backoffBase(1), backoffBase(3))
	}
	if got := backoffBase(1000); got <= 0 {
		t.Fatalf("backoffBase(1000) = %v, want a positive delay", got)
	}
}
