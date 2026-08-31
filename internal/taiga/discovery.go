package taiga

import (
	"context"
	"encoding/json"
	"errors"
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
	if resp.StatusCode != http.StatusOK {
		return FrontConfig{}, decodeAPIError("GET "+confURL.Path, resp.StatusCode, data)
	}
	var config FrontConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return FrontConfig{}, fmt.Errorf("decode Taiga frontend config: %w", err)
	}
	if config.API == "" {
		return FrontConfig{}, errors.New("taiga frontend config does not declare an API URL")
	}
	config.API, err = NormalizeAPIURL(config.API)
	if err != nil {
		return FrontConfig{}, err
	}
	return config, nil
}
