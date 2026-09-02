package taiga

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// blockingServer answers nothing until the test ends: the shape of a peer
// that has stopped responding. answer runs first and reports whether it
// answered the request in full, in which case the handler does not block.
func blockingServer(t *testing.T, answer func(http.ResponseWriter, *http.Request) bool) *httptest.Server {
	t.Helper()
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if answer != nil && answer(w, r) {
			return
		}
		<-release
	}))
	// Close waits for handlers to return, so the handlers are released first.
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) })
	return server
}

// A JSON request Taiga does not answer in time is one attempt gone, not a
// process that waits forever. For a read that is retryable; for a write the
// outcome is unknown, since the request may have arrived.
func TestJSONRequestAttemptIsBounded(t *testing.T) {
	server := blockingServer(t, nil)
	client, err := NewClient(server.URL+"/api/v1/", WithMaxRetries(1))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.requestTimeout = 50 * time.Millisecond

	var out any
	_, err = client.Get(context.Background(), "projects", nil, &out)
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Kind != KindTransport || !apiErr.Retryable {
		t.Fatalf("unanswered read reported %v, want a retryable %s", err, KindTransport)
	}
	if errors.Is(err, context.Canceled) {
		t.Error("nobody cancelled anything; the deadline must not read as an interrupt")
	}
	_, err = client.Post(context.Background(), "issues", map[string]any{"subject": "x"}, nil)
	if !errors.As(err, &apiErr) || apiErr.Kind != KindAmbiguousCommit {
		t.Fatalf("unanswered write reported %v, want %s", err, KindAmbiguousCommit)
	}
}

// A transfer is as long as the file and the link make it. The request
// timeout bounds a JSON exchange, not a download or an upload.
func TestTransferOutlivesTheRequestTimeout(t *testing.T) {
	const chunks, chunkSize, pace = 20, 1024, 10 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/issues/attachments":
			buffer := make([]byte, chunkSize)
			for {
				if _, err := io.ReadFull(r.Body, buffer); err != nil {
					break
				}
				time.Sleep(pace)
			}
			_, _ = io.WriteString(w, `{"id":3,"name":"a.bin","size":1}`)
		default:
			flusher := w.(http.Flusher)
			for i := 0; i < chunks; i++ {
				_, _ = w.Write(make([]byte, chunkSize))
				flusher.Flush()
				time.Sleep(pace)
			}
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL + "/api/v1/")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.requestTimeout = 50 * time.Millisecond

	result, err := client.DownloadAttachment(context.Background(), Attachment{URL: server.URL + "/media/big"}, io.Discard)
	if err != nil {
		t.Fatalf("a %v download failed under a %v request timeout: %v", chunks*pace, client.requestTimeout, err)
	}
	if result.Bytes != chunks*chunkSize {
		t.Errorf("downloaded %d bytes, want %d", result.Bytes, chunks*chunkSize)
	}
	_, err = client.CreateAttachment(context.Background(), "issue", 1, 2, "a.bin", "", false, strings.NewReader(strings.Repeat("x", chunks*chunkSize)))
	if err != nil {
		t.Fatalf("a slow upload failed under a %v request timeout: %v", client.requestTimeout, err)
	}
}

// A transfer that stops moving is abandoned and says so, since neither a
// dead peer nor a stalled link would otherwise ever end it.
func TestStalledTransferIsAbandoned(t *testing.T) {
	server := blockingServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodGet {
			// One byte gets through, then nothing.
			_, _ = w.Write([]byte("x"))
			w.(http.Flusher).Flush()
		}
		return false
	})
	client, err := NewClient(server.URL + "/api/v1/")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.stallTimeout = 100 * time.Millisecond

	outcome := make(chan error, 1)
	go func() {
		_, err := client.DownloadAttachment(context.Background(), Attachment{URL: server.URL + "/media/stalled"}, io.Discard)
		outcome <- err
	}()
	err = awaitOutcome(t, outcome)
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Kind != KindTransport {
		t.Fatalf("stalled download reported %v, want %s", err, KindTransport)
	}
	if !strings.HasPrefix(apiErr.Message, "no data moved for 100ms; ") {
		t.Errorf("message = %q, want it to name the stall", apiErr.Message)
	}

	go func() {
		_, err := client.CreateAttachment(context.Background(), "issue", 1, 2, "a.bin", "", false, strings.NewReader("payload"))
		outcome <- err
	}()
	err = awaitOutcome(t, outcome)
	if !errors.As(err, &apiErr) || apiErr.Kind != KindAmbiguousCommit {
		t.Fatalf("stalled upload reported %v, want %s", err, KindAmbiguousCommit)
	}
	if !strings.HasPrefix(apiErr.Message, "no data moved for 100ms; ") {
		t.Errorf("message = %q, want it to name the stall", apiErr.Message)
	}
}

func awaitOutcome(t *testing.T, outcome <-chan error) error {
	t.Helper()
	select {
	case err := <-outcome:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("the transfer did not end; a stalled peer would hang the command")
		return nil
	}
}

// A refresh Taiga never answers is one more way the refresh did not reach
// Taiga, and is reported as such. It is not the operator running out of
// patience, which is the only thing a bare deadline error would mean to the
// command.
func TestRefreshThatTimesOutIsATransportFailure(t *testing.T) {
	server := blockingServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/api/v1/users/me" {
			return false
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"detail":"Token is expired"}`)
		return true
	})
	client, err := NewClient(server.URL+"/api/v1/", WithToken("old"), WithRefreshToken("still-good", func(string, string) error { return nil }))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.requestTimeout = 50 * time.Millisecond

	_, err = client.Me(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Kind != KindTransport || !apiErr.Retryable {
		t.Fatalf("unanswered refresh reported %v, want a retryable %s", err, KindTransport)
	}
	if apiErr.Operation != "POST /api/v1/auth/refresh" {
		t.Errorf("operation = %q, want the refresh", apiErr.Operation)
	}
}

// Stopping a download after the first byte is the same interruption as
// stopping it before, and is reported as one rather than as a fault to retry.
func TestDownloadInterruptedMidStreamIsAnInterrupt(t *testing.T) {
	firstByte := make(chan struct{})
	server := blockingServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		_, _ = w.Write([]byte("x"))
		w.(http.Flusher).Flush()
		close(firstByte)
		return false
	})
	client, err := NewClient(server.URL + "/api/v1/")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-firstByte
		cancel()
	}()

	_, err = client.DownloadAttachment(ctx, Attachment{URL: server.URL + "/media/partial"}, io.Discard)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		t.Errorf("an interrupt is not a %s to retry", apiErr.Kind)
	}
}
