package taiga

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type FrontConfig struct {
	API       string `json:"api"`
	EventsURL string `json:"eventsUrl"`
	BaseHref  string `json:"baseHref"`
}

func DiscoverAPI(ctx context.Context, httpClient *http.Client, host string) (FrontConfig, error) {
	host = strings.TrimSpace(host)
	parsed, err := url.Parse(host)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return FrontConfig{}, fmt.Errorf("invalid Taiga host %q", host)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return FrontConfig{}, fmt.Errorf("unsupported Taiga host scheme %q", parsed.Scheme)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.User = nil
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	confURL := parsed.ResolveReference(&url.URL{Path: "conf.json"})
	// The client carries no deadline of its own, and conf.json is one small
	// file, so this bounds it the way a JSON request is bounded.
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, confURL.String(), nil)
	if err != nil {
		return FrontConfig{}, fmt.Errorf("create discovery request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return FrontConfig{}, &Error{Kind: KindTransport, Operation: "discover Taiga API", Message: "Taiga frontend is unavailable", Retryable: true, Cause: err}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	closeErr := resp.Body.Close()
	if err != nil {
		return FrontConfig{}, fmt.Errorf("read Taiga frontend config: %w", err)
	}
	if closeErr != nil {
		return FrontConfig{}, fmt.Errorf("close Taiga frontend response: %w", closeErr)
	}
	// Whatever went wrong, the person reading it typed a host, and what they
	// need to know is which URL was tried and what came back: a forum or a
	// marketing site answers 404 or a page of HTML, and "Not Found" on its
	// own says nothing about either.
	if resp.StatusCode != http.StatusOK {
		apiErr := decodeAPIError("GET "+confURL.Path, resp.StatusCode, data)
		apiErr.Message = notFrontendMessage(confURL, fmt.Sprintf("it answered %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)))
		return FrontConfig{}, apiErr
	}
	var config FrontConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return FrontConfig{}, &Error{Kind: KindValidation, Operation: "GET " + confURL.Path, Message: notFrontendMessage(confURL, "the answer is not JSON"), Retryable: false, Cause: err}
	}
	if config.API == "" {
		return FrontConfig{}, &Error{Kind: KindValidation, Operation: "GET " + confURL.Path, Message: notFrontendMessage(confURL, "it declares no API URL"), Retryable: false}
	}
	config.API, err = NormalizeAPIURL(config.API)
	if err != nil {
		return FrontConfig{}, err
	}
	return config, nil
}

// notFrontendMessage explains a host that did not turn out to be a Taiga web
// app, naming the URL that was tried so the reader can see the mistake.
func notFrontendMessage(confURL *url.URL, reason string) string {
	return fmt.Sprintf("%s is not a Taiga frontend configuration: %s; --host is the address of the Taiga web app, such as https://tree.taiga.io/", confURL, reason)
}
