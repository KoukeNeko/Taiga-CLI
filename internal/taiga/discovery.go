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

const (
	// hostedTaigaDomain is the domain of the Taiga its makers run. Its web app
	// lives at one address, and every other name under the domain is a forum,
	// a site or the API, so a host there that is not the app can be told where
	// the app is as a fact rather than as an example.
	hostedTaigaDomain = "taiga.io"
	hostedTaigaApp    = "https://tree.taiga.io/"
	// maxDiscoveryBytes bounds what a probe reads: a frontend configuration
	// and a locale list are both a few kilobytes.
	maxDiscoveryBytes = 1 << 20
)

// discoveryAttempt is one URL discovery tried and what it found there.
type discoveryAttempt struct {
	url     *url.URL
	outcome string
}

// DiscoverAPI finds the Taiga API behind host. host is the address of the
// Taiga web app, whose conf.json names the API. Failing that, the address
// itself and its api/v1/ path are tried as the API, because people pass the
// API's address here too. Nothing beyond the origin that was typed is
// contacted: the right address cannot be derived from a wrong one, only
// guessed, and a guess at another host is a request nobody asked for.
func DiscoverAPI(ctx context.Context, httpClient *http.Client, host string) (FrontConfig, error) {
	parsed, err := parseHost(host)
	if err != nil {
		return FrontConfig{}, err
	}
	// One budget for every probe, the way a JSON request is bounded.
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()
	confURL := parsed.ResolveReference(&url.URL{Path: "conf.json"})
	status, data, err := fetchDiscovery(ctx, httpClient, confURL)
	if err != nil {
		return FrontConfig{}, &Error{Kind: KindTransport, Operation: "discover Taiga API", Message: "Taiga frontend is unavailable", Retryable: true, Cause: err}
	}
	config, outcome := parseFrontConfig(status, data)
	if outcome == "" {
		return config, nil
	}
	tried := []discoveryAttempt{{url: confURL, outcome: outcome}}
	for _, base := range apiCandidates(parsed) {
		localesURL := base.ResolveReference(&url.URL{Path: "locales"})
		found, outcome := probeAPI(ctx, httpClient, localesURL)
		if found {
			api, err := NormalizeAPIURL(base.String())
			if err != nil {
				return FrontConfig{}, err
			}
			return FrontConfig{API: api}, nil
		}
		tried = append(tried, discoveryAttempt{url: localesURL, outcome: outcome})
	}
	return FrontConfig{}, notTaigaError(parsed, confURL, status, data, tried)
}

func parseHost(host string) (*url.URL, error) {
	host = strings.TrimSpace(host)
	parsed, err := url.Parse(host)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid Taiga host %q", host)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported Taiga host scheme %q", parsed.Scheme)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.User = nil
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	return parsed, nil
}

// apiCandidates lists where the API may be when the host is not the web app:
// at the address itself, and under api/v1/ unless the address is that already.
func apiCandidates(host *url.URL) []*url.URL {
	candidates := []*url.URL{host}
	if !strings.HasSuffix(host.Path, "/api/v1/") {
		candidates = append(candidates, host.ResolveReference(&url.URL{Path: "api/v1/"}))
	}
	return candidates
}

func fetchDiscovery(ctx context.Context, httpClient *http.Client, target *url.URL) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return 0, nil, fmt.Errorf("create discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxDiscoveryBytes))
	closeErr := resp.Body.Close()
	if readErr != nil {
		return 0, nil, fmt.Errorf("read discovery response: %w", readErr)
	}
	if closeErr != nil {
		return 0, nil, fmt.Errorf("close discovery response: %w", closeErr)
	}
	return resp.StatusCode, data, nil
}

// parseFrontConfig reads a conf.json answer. The outcome is empty when the
// answer is a usable configuration, and otherwise says what came back instead.
func parseFrontConfig(status int, data []byte) (FrontConfig, string) {
	if status != http.StatusOK {
		return FrontConfig{}, answered(status)
	}
	var config FrontConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return FrontConfig{}, "answered with something other than JSON"
	}
	if config.API == "" {
		return FrontConfig{}, "answered JSON that declares no API URL"
	}
	api, err := NormalizeAPIURL(config.API)
	if err != nil {
		return FrontConfig{}, "declares an API URL that cannot be used: " + err.Error()
	}
	config.API = api
	return config, ""
}

// probeAPI reports whether localesURL is Taiga's locale list, which every
// Taiga API serves without a credential. A web app's server answers unknown
// paths with its page, so only a JSON list of locales counts.
func probeAPI(ctx context.Context, httpClient *http.Client, localesURL *url.URL) (bool, string) {
	status, data, err := fetchDiscovery(ctx, httpClient, localesURL)
	if err != nil {
		return false, "could not be reached: " + err.Error()
	}
	if status != http.StatusOK {
		return false, answered(status)
	}
	var locales []struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(data, &locales); err != nil {
		return false, "answered with something other than JSON"
	}
	if len(locales) == 0 || locales[0].Code == "" {
		return false, "answered JSON that is not a Taiga locale list"
	}
	return true, ""
}

func answered(status int) string {
	return fmt.Sprintf("answered %d %s", status, http.StatusText(status))
}

// notTaigaError reports a host that is neither a Taiga web app nor its API.
// The kind follows what conf.json answered, so that a 404 stays not_found and
// a page where JSON was expected is validation; the message lists every URL
// tried, because the person who typed the host needs to see where it led.
func notTaigaError(host, confURL *url.URL, confStatus int, confData []byte, tried []discoveryAttempt) *Error {
	attempts := make([]string, 0, len(tried))
	for _, attempt := range tried {
		attempts = append(attempts, attempt.url.String()+" "+attempt.outcome)
	}
	message := fmt.Sprintf("%s is not the address of a Taiga web app or API: %s. %s", host, strings.Join(attempts, "; "), discoveryAdvice(host.Hostname()))
	if confStatus != http.StatusOK {
		apiErr := decodeAPIError("GET "+confURL.Path, confStatus, confData)
		apiErr.Message = message
		return apiErr
	}
	return &Error{Kind: KindValidation, Operation: "GET " + confURL.Path, Message: message, Retryable: false}
}

// discoveryAdvice says what --host should have been. Under the hosted Taiga's
// domain the web app has one known address, which is stated; elsewhere it can
// only be described.
func discoveryAdvice(hostname string) string {
	hosted := hostname == hostedTaigaDomain || strings.HasSuffix(hostname, "."+hostedTaigaDomain)
	if hosted && "https://"+hostname+"/" != hostedTaigaApp {
		return "The hosted Taiga web app is " + hostedTaigaApp
	}
	return "--host is the address of the Taiga web app, such as " + hostedTaigaApp
}
