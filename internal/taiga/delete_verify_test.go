package taiga

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// deleteVerifyServer answers the DELETE with 204 and the read-back GET with
// verifyStatus, which is the whole of what a delete-then-verify exchanges.
func deleteVerifyServer(t *testing.T, path string, verifyStatus int) *Client {
	t.Helper()
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
			http.Error(w, `{"detail":"unexpected"}`, http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodDelete:
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			if !deleted {
				t.Error("the record was read back before it was deleted")
			}
			w.WriteHeader(verifyStatus)
			_, _ = w.Write([]byte(`{"id":1}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL+"/api/v1/", WithMaxRetries(1))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

// Every delete that reads the record back reports the same two shapes of
// doubt, so that a caller can treat them alike: Taiga still serving the
// record, and Taiga failing to say either way. Each is pinned per endpoint
// because they were written separately and the wording is what a person
// reads when a deletion is in question.
func TestDeleteReadsTheRecordBackBeforeReportingSuccess(t *testing.T) {
	moveTo := int64(2)
	for _, testCase := range []struct {
		name        string
		path        string
		stillServed string
		unanswered  string
		remove      func(*Client, context.Context) error
	}{
		{
			name: "work item", path: "/api/v1/issues/1",
			stillServed: "Taiga still returns the work item after deletion",
			unanswered:  "work item was deleted but could not be verified",
			remove:      func(c *Client, ctx context.Context) error { return c.DeleteWorkItem(ctx, "issue", 1) },
		},
		{
			name: "milestone", path: "/api/v1/milestones/1",
			stillServed: "Taiga still returns the sprint after deletion",
			unanswered:  "sprint was deleted but could not be verified",
			remove:      func(c *Client, ctx context.Context) error { return c.DeleteMilestone(ctx, 1) },
		},
		{
			name: "wiki link", path: "/api/v1/wiki-links/1",
			stillServed: "Taiga still returns the wiki link after deletion",
			unanswered:  "wiki link was deleted but could not be verified",
			remove:      func(c *Client, ctx context.Context) error { return c.DeleteWikiLink(ctx, 1) },
		},
		{
			name: "workflow metadata", path: "/api/v1/issue-statuses/1",
			stillServed: "Taiga still returns the metadata after deletion",
			unanswered:  "metadata was deleted but could not be verified",
			remove: func(c *Client, ctx context.Context) error {
				return c.DeleteWorkflowMetadata(ctx, "issue-status", 1, moveTo)
			},
		},
		{
			name: "due date", path: "/api/v1/issue-due-dates/1",
			stillServed: "Taiga still returns the resource after deletion",
			unanswered:  "resource was deleted but could not be verified",
			remove:      func(c *Client, ctx context.Context) error { return c.DeleteDueDate(ctx, "issue", 1) },
		},
		{
			name: "swimlane", path: "/api/v1/swimlanes/1",
			stillServed: "Taiga still returns the resource after deletion",
			unanswered:  "resource was deleted but could not be verified",
			remove:      func(c *Client, ctx context.Context) error { return c.DeleteSwimlane(ctx, 1, &moveTo) },
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Run("gone", func(t *testing.T) {
				client := deleteVerifyServer(t, testCase.path, http.StatusNotFound)
				if err := testCase.remove(client, context.Background()); err != nil {
					t.Fatalf("a record Taiga no longer serves is deleted, got %v", err)
				}
			})
			t.Run("still served", func(t *testing.T) {
				client := deleteVerifyServer(t, testCase.path, http.StatusOK)
				err := testCase.remove(client, context.Background())
				var apiErr *Error
				if !errors.As(err, &apiErr) || apiErr.Kind != KindAmbiguousCommit {
					t.Fatalf("got %v, want %s", err, KindAmbiguousCommit)
				}
				if apiErr.Message != testCase.stillServed {
					t.Errorf("message = %q, want %q", apiErr.Message, testCase.stillServed)
				}
			})
			t.Run("unanswered", func(t *testing.T) {
				client := deleteVerifyServer(t, testCase.path, http.StatusInternalServerError)
				err := testCase.remove(client, context.Background())
				var apiErr *Error
				if !errors.As(err, &apiErr) || apiErr.Kind != KindAmbiguousCommit {
					t.Fatalf("got %v, want %s", err, KindAmbiguousCommit)
				}
				if apiErr.Message != testCase.unanswered {
					t.Errorf("message = %q, want %q", apiErr.Message, testCase.unanswered)
				}
				if apiErr.Cause == nil {
					t.Error("the error must carry why the read-back failed")
				}
			})
		})
	}
}
