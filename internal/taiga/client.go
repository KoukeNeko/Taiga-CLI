package taiga

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxResponseBytes = 16 << 20

const (
	backoffStep = 100 * time.Millisecond
	// jitterWindow spreads retries so that CLI invocations throttled by the
	// same response do not resend in lockstep.
	jitterWindow = 100 * time.Millisecond
	// maxBackoffShift keeps the doubling below the point where the shift would
	// overflow time.Duration and produce a negative delay.
	maxBackoffShift = 20
	maxRetryAfter   = 30 * time.Second
)

type Client struct {
	baseURL      *url.URL
	httpClient   *http.Client
	token        string
	refreshToken string
	onRefresh    func(string, string) error
	verbose      io.Writer
	maxRetries   int
	sleep        func(context.Context, time.Duration) error
}

type ClientOption func(*Client)

func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = client }
}

func WithToken(token string) ClientOption {
	return func(c *Client) { c.token = strings.TrimSpace(token) }
}

func WithRefreshToken(refreshToken string, onRefresh func(string, string) error) ClientOption {
	return func(c *Client) {
		c.refreshToken = strings.TrimSpace(refreshToken)
		c.onRefresh = onRefresh
	}
}

func WithVerbose(writer io.Writer) ClientOption {
	return func(c *Client) { c.verbose = writer }
}

func WithMaxRetries(retries int) ClientOption {
	return func(c *Client) { c.maxRetries = retries }
}

func NewClient(rawURL string, options ...ClientOption) (*Client, error) {
	normalized, err := NormalizeAPIURL(rawURL)
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return nil, fmt.Errorf("parse API URL: %w", err)
	}
	client := &Client{
		baseURL:    parsed,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		maxRetries: 3,
		sleep: func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-timer.C:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
	for _, option := range options {
		option(client)
	}
	return client, nil
}

func NormalizeAPIURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("API URL cannot be empty")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid API URL %q", raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported API URL scheme %q", parsed.Scheme)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.User = nil
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	return parsed.String(), nil
}

func (c *Client) APIURL() string { return c.baseURL.String() }

func (c *Client) SetToken(token string) { c.token = strings.TrimSpace(token) }

func (c *Client) Get(ctx context.Context, path string, query url.Values, out any) (http.Header, error) {
	return c.doJSON(ctx, http.MethodGet, path, query, nil, out, true, false)
}

// GetOnce performs a GET without automatic retries. It is reserved for Taiga
// endpoints that use GET to enqueue work and are therefore not idempotent.
func (c *Client) GetOnce(ctx context.Context, path string, query url.Values, out any) (http.Header, error) {
	return c.doJSON(ctx, http.MethodGet, path, query, nil, out, false, true)
}

func (c *Client) Post(ctx context.Context, path string, body, out any) (http.Header, error) {
	return c.doJSON(ctx, http.MethodPost, path, nil, body, out, false, false)
}

func (c *Client) PostQuery(ctx context.Context, path string, query url.Values, body, out any) (http.Header, error) {
	return c.doJSON(ctx, http.MethodPost, path, query, body, out, false, false)
}

func (c *Client) Patch(ctx context.Context, path string, body, out any) (http.Header, error) {
	return c.doJSON(ctx, http.MethodPatch, path, nil, body, out, false, false)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil, false, false)
	return err
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body, out any, retryGET, ambiguousGET bool) (http.Header, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
	}
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: strings.TrimPrefix(path, "/")})
	endpoint.RawQuery = query.Encode()
	attempts := 1
	if method == http.MethodGet && retryGET {
		attempts = c.maxRetries
		if attempts < 1 {
			attempts = 1
		}
	}
	refreshed := false
	for attempt := 1; attempt <= attempts; attempt++ {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "aihki/0.1")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		started := time.Now()
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			c.log(method, endpoint.Path, 0, time.Since(started))
			if method != http.MethodGet || ambiguousGET {
				return nil, &Error{Kind: KindAmbiguousCommit, Operation: method + " " + endpoint.Path, Message: "request may have been committed; verify before retrying", Retryable: false, Cause: err}
			}
			if attempt < attempts {
				if err := c.sleep(ctx, retryDelay(attempt, "")); err != nil {
					return nil, err
				}
				continue
			}
			return nil, &Error{Kind: KindTransport, Operation: method + " " + endpoint.Path, Message: "Taiga API is unavailable", Retryable: true, Cause: err}
		}
		c.log(method, endpoint.Path, resp.StatusCode, time.Since(started))
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		closeErr := resp.Body.Close()
		if readErr != nil {
			if method != http.MethodGet || ambiguousGET {
				return nil, &Error{Kind: KindAmbiguousCommit, Operation: method + " " + endpoint.Path, Message: "request may have been committed; verify before retrying", Retryable: false, Cause: readErr}
			}
			return nil, &Error{Kind: KindTransport, Operation: method + " " + endpoint.Path, Message: "read Taiga API response", Retryable: method == http.MethodGet, Cause: readErr}
		}
		if closeErr != nil {
			if method != http.MethodGet || ambiguousGET {
				return nil, &Error{Kind: KindAmbiguousCommit, Operation: method + " " + endpoint.Path, Message: "request may have been committed; verify before retrying", Retryable: false, Cause: closeErr}
			}
			return nil, &Error{Kind: KindTransport, Operation: method + " " + endpoint.Path, Message: "close Taiga API response", Retryable: false, Cause: closeErr}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			apiErr := decodeAPIError(method+" "+endpoint.Path, resp.StatusCode, data)
			if resp.StatusCode == http.StatusUnauthorized && !refreshed && c.refreshToken != "" && !strings.HasSuffix(endpoint.Path, "/auth/refresh") {
				if err := c.refresh(ctx); err == nil {
					refreshed = true
					attempt--
					continue
				}
			}
			if method == http.MethodGet && attempt < attempts && retryableStatus(resp.StatusCode) {
				if err := c.sleep(ctx, retryDelay(attempt, resp.Header.Get("Retry-After"))); err != nil {
					return nil, err
				}
				continue
			}
			return resp.Header, apiErr
		}
		if out != nil && len(bytes.TrimSpace(data)) > 0 {
			if err := json.Unmarshal(data, out); err != nil {
				if method != http.MethodGet || ambiguousGET {
					return resp.Header, &Error{Kind: KindAmbiguousCommit, Operation: method + " " + endpoint.Path, Message: "request was accepted but its result could not be decoded; verify before retrying", Retryable: false, Cause: err}
				}
				return resp.Header, &Error{Kind: KindTransport, Operation: method + " " + endpoint.Path, Message: "decode Taiga API response", Retryable: false, Cause: err}
			}
		}
		return resp.Header, nil
	}
	return nil, &Error{Kind: KindTransport, Message: "Taiga API request failed", Retryable: true}
}

func (c *Client) refresh(ctx context.Context) error {
	endpoint := c.baseURL.ResolveReference(&url.URL{Path: "auth/refresh"})
	payload, err := json.Marshal(map[string]string{"refresh": c.refreshToken})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "aihki/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	closeErr := resp.Body.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError("POST "+endpoint.Path, resp.StatusCode, data)
	}
	var response AuthResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}
	if response.AuthToken == "" {
		return errors.New("taiga refresh response did not contain an auth token")
	}
	c.token = response.AuthToken
	if response.RefreshToken != "" {
		c.refreshToken = response.RefreshToken
	}
	if c.onRefresh != nil {
		return c.onRefresh(c.token, c.refreshToken)
	}
	return nil
}

func decodeAPIError(operation string, status int, data []byte) *Error {
	details := map[string]any{}
	_ = json.Unmarshal(data, &details)
	message := http.StatusText(status)
	for _, key := range []string{"_error_message", "detail", "message"} {
		if value, ok := details[key].(string); ok && strings.TrimSpace(value) != "" {
			message = value
			break
		}
	}
	if message == http.StatusText(status) {
		// Taiga reports field validation, including a stale version, as a bare
		// map of field names to explanations with none of the keys above. Left
		// alone that surfaces to a person as "Bad Request", which says nothing
		// about what to do next.
		if explanation := fieldExplanation(details); explanation != "" {
			message = explanation
		}
	}
	kind := KindValidation
	retryable := false
	switch status {
	case http.StatusUnauthorized:
		kind = KindAuth
	case http.StatusForbidden:
		kind = KindForbidden
	case http.StatusNotFound:
		kind = KindNotFound
	case http.StatusConflict:
		kind = KindConflict
	case http.StatusTooManyRequests:
		kind, retryable = KindThrottled, true
	default:
		if status >= 500 {
			kind, retryable = KindTransport, true
		}
	}
	if errorType, _ := details["_error_type"].(string); strings.Contains(strings.ToLower(errorType), "version") {
		kind = KindConflict
	}
	if strings.Contains(strings.ToLower(message), "version") && status == http.StatusBadRequest {
		kind = KindConflict
	}
	if _, ok := details["version"]; ok && status == http.StatusBadRequest {
		kind = KindConflict
	}
	return &Error{Kind: kind, Operation: operation, Message: message, Retryable: retryable, UpstreamStatus: status, Details: details}
}

// fieldExplanation renders a Taiga field-validation body as a sentence. Keys
// are sorted so the same response always produces the same message.
func fieldExplanation(details map[string]any) string {
	fields := make([]string, 0, len(details))
	for key := range details {
		if strings.HasPrefix(key, "_") {
			continue
		}
		if _, ok := details[key].(string); ok {
			fields = append(fields, key)
		}
	}
	sort.Strings(fields)
	parts := make([]string, 0, len(fields))
	for _, key := range fields {
		parts = append(parts, key+": "+details[key].(string))
	}
	return strings.Join(parts, "; ")
}

func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func retryDelay(attempt int, retryAfter string) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds > 0 {
		return min(time.Duration(seconds)*time.Second, maxRetryAfter)
	}
	if date, err := http.ParseTime(strings.TrimSpace(retryAfter)); err == nil {
		return min(max(time.Until(date), 0), maxRetryAfter)
	}
	return backoffBase(attempt) + rand.N(jitterWindow)
}

func backoffBase(attempt int) time.Duration {
	if attempt > maxBackoffShift {
		attempt = maxBackoffShift
	}
	return backoffStep * time.Duration(1<<(attempt-1))
}

func (c *Client) log(method, path string, status int, elapsed time.Duration) {
	if c.verbose == nil {
		return
	}
	if status == 0 {
		_, _ = fmt.Fprintf(c.verbose, "%s %s transport-error %s\n", method, path, elapsed.Round(time.Millisecond))
		return
	}
	_, _ = fmt.Fprintf(c.verbose, "%s %s %d %s\n", method, path, status, elapsed.Round(time.Millisecond))
}

func pageFromHeaders(header http.Header, fallbackSize int) Page {
	integer := func(name string) int {
		value, _ := strconv.Atoi(header.Get(name))
		return value
	}
	page := Page{
		Number: integer("X-Pagination-Current"),
		Size:   integer("X-Paginated-By"),
		Total:  integer("X-Pagination-Count"),
		Next:   integer("X-Pagination-Next"),
		Prev:   integer("X-Pagination-Prev"),
	}
	if page.Number == 0 {
		page.Number = 1
	}
	if page.Size == 0 {
		page.Size = fallbackSize
	}
	return page
}
