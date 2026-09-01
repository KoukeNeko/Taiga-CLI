package taiga

import (
	"bytes"
	"context"
	"crypto/sha1" // #nosec G505 -- test fixture for Taiga's compatibility digest.
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloadAttachmentStreamsAndVerifiesHashes(t *testing.T) {
	content := []byte("attachment evidence")
	sha1Sum := sha1.Sum(content) // #nosec G401 -- test fixture.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/media/evidence.txt" || r.URL.Query().Get("token") != "signed" || r.Header.Get("Authorization") != "" {
			t.Fatalf("request=%s auth=%q", r.URL.String(), r.Header.Get("Authorization"))
		}
		_, _ = w.Write(content)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/api/v1/", WithToken("bearer-must-not-cross-media-boundary"))
	attachment := Attachment{Name: "evidence.txt", Size: int64(len(content)), URL: server.URL + "/media/evidence.txt?token=signed#refresh=issue:1", SHA1: hex.EncodeToString(sha1Sum[:])}
	var output bytes.Buffer
	result, err := client.DownloadAttachment(context.Background(), attachment, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), content) || result.Bytes != int64(len(content)) || result.SHA1 != attachment.SHA1 || len(result.SHA256) != 64 {
		t.Fatalf("result=%#v output=%q", result, output.Bytes())
	}
}

func TestWikiLinkCRUDAndVerifiedDelete(t *testing.T) {
	link := WikiLink{ID: 3, Project: 1, Title: "Guide", Href: "guide", Order: 10}
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/wiki-links":
			if r.URL.Query().Get("project") != "1" {
				t.Fatalf("query=%s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode([]WikiLink{link})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/wiki-links":
			_ = json.NewEncoder(w).Encode(link)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/wiki-links/3":
			link.Title = "Updated"
			_ = json.NewEncoder(w).Encode(link)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/wiki-links/3":
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/wiki-links/3" && deleted:
			http.Error(w, `{"detail":"Not found"}`, http.StatusNotFound)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/wiki-links/3":
			_ = json.NewEncoder(w).Encode(link)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL + "/api/v1/")
	links, _, err := client.ListWikiLinks(context.Background(), 1, 1, 30)
	if err != nil || len(links) != 1 {
		t.Fatalf("links=%#v err=%v", links, err)
	}
	if _, err := client.CreateWikiLink(context.Background(), CreateWikiLinkRequest{Project: 1, Title: "Guide"}); err != nil {
		t.Fatal(err)
	}
	title := "Updated"
	updated, err := client.UpdateWikiLink(context.Background(), 3, UpdateWikiLinkRequest{Title: &title})
	if err != nil || updated.Title != "Updated" {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
	if err := client.DeleteWikiLink(context.Background(), 3); err != nil {
		t.Fatal(err)
	}
}
