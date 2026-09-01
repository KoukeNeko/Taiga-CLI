package taiga

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type NotifyPolicy struct {
	ID              int64  `json:"id"`
	Project         int64  `json:"project"`
	ProjectName     string `json:"project_name"`
	NotifyLevel     int    `json:"notify_level"`
	LiveNotifyLevel int    `json:"live_notify_level"`
	WebNotifyLevel  bool   `json:"web_notify_level"`
}

func (c *Client) ListNotifyPolicies(ctx context.Context) ([]NotifyPolicy, error) {
	var values []NotifyPolicy
	_, err := c.Get(ctx, "notify-policies", url.Values{"page_size": {"1000"}}, &values)
	return values, err
}

func (c *Client) GetNotifyPolicy(ctx context.Context, id int64) (NotifyPolicy, error) {
	var value NotifyPolicy
	_, err := c.Get(ctx, fmt.Sprintf("notify-policies/%d", id), nil, &value)
	return value, err
}

func (c *Client) CreateNotifyPolicy(ctx context.Context, projectID, userID int64, notify, live int, web bool) (NotifyPolicy, error) {
	var value NotifyPolicy
	body := map[string]any{"project": projectID, "user": userID, "notify_level": notify, "live_notify_level": live, "web_notify_level": web}
	_, err := c.Post(ctx, "notify-policies", body, &value)
	return value, err
}

func (c *Client) UpdateNotifyPolicy(ctx context.Context, id int64, fields map[string]any) (NotifyPolicy, error) {
	var value NotifyPolicy
	_, err := c.Patch(ctx, fmt.Sprintf("notify-policies/%d", id), fields, &value)
	return value, err
}

func (c *Client) DeleteNotifyPolicy(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("notify-policies/%d", id))
}

type WebNotification struct {
	ID        int64          `json:"id"`
	EventType int            `json:"event_type"`
	User      int64          `json:"user"`
	Data      map[string]any `json:"data"`
	Created   string         `json:"created"`
	Read      *string        `json:"read"`
}

func (c *Client) ListWebNotifications(ctx context.Context, unread bool, page, limit int) ([]WebNotification, int, error) {
	query := url.Values{"page": {fmt.Sprint(page)}, "page_size": {fmt.Sprint(limit)}}
	if unread {
		query.Set("only_unread", "true")
	}
	var result struct {
		Objects []WebNotification `json:"objects"`
		Total   int               `json:"total"`
	}
	_, err := c.Get(ctx, "web-notifications", query, &result)
	return result.Objects, result.Total, err
}

func (c *Client) MarkWebNotificationRead(ctx context.Context, id int64) error {
	_, err := c.Patch(ctx, fmt.Sprintf("web-notifications/%d/set-as-read", id), map[string]any{}, nil)
	return err
}

func (c *Client) MarkAllWebNotificationsRead(ctx context.Context) error {
	_, err := c.Post(ctx, "web-notifications/set-as-read", map[string]any{}, nil)
	return err
}

type Application struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Web         string `json:"web"`
	Description string `json:"description"`
	IconURL     string `json:"icon_url"`
}
type ApplicationToken struct {
	ID          int64       `json:"id"`
	User        int64       `json:"user"`
	Application Application `json:"application"`
	AuthCode    string      `json:"auth_code,omitempty"`
	NextURL     string      `json:"next_url,omitempty"`
}

func (c *Client) ListApplications(ctx context.Context) ([]Application, error) {
	tokens, err := c.ListApplicationTokens(ctx, nil)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	values := make([]Application, 0, len(tokens))
	for _, token := range tokens {
		if token.Application.ID == "" || seen[token.Application.ID] {
			continue
		}
		seen[token.Application.ID] = true
		values = append(values, token.Application)
	}
	return values, nil
}

func (c *Client) ListApplicationTokens(ctx context.Context, applicationID *string) ([]ApplicationToken, error) {
	query := url.Values{"page_size": {"1000"}}
	if applicationID != nil && *applicationID != "" {
		query.Set("application", *applicationID)
	}
	var values []ApplicationToken
	_, err := c.Get(ctx, "application-tokens", query, &values)
	return values, err
}

func (c *Client) RevokeApplicationToken(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("application-tokens/%d", id))
}

type StorageEntry struct {
	Key          string `json:"key"`
	Value        any    `json:"value"`
	CreatedDate  string `json:"created_date,omitempty"`
	ModifiedDate string `json:"modified_date,omitempty"`
}

func (c *Client) ListStorage(ctx context.Context) ([]StorageEntry, error) {
	var values []StorageEntry
	_, err := c.Get(ctx, "user-storage", url.Values{"page_size": {"1000"}}, &values)
	return values, err
}

func (c *Client) GetStorage(ctx context.Context, key string) (StorageEntry, error) {
	var value StorageEntry
	_, err := c.Get(ctx, "user-storage/"+url.PathEscape(key), nil, &value)
	return value, err
}

func (c *Client) CreateStorage(ctx context.Context, key string, value any) (StorageEntry, error) {
	var result StorageEntry
	_, err := c.Post(ctx, "user-storage", map[string]any{"key": key, "value": value}, &result)
	return result, err
}

func (c *Client) UpdateStorage(ctx context.Context, key string, value any) (StorageEntry, error) {
	var result StorageEntry
	_, err := c.Patch(ctx, "user-storage/"+url.PathEscape(key), map[string]any{"value": value}, &result)
	return result, err
}

func (c *Client) DeleteStorage(ctx context.Context, key string) error {
	return c.Delete(ctx, "user-storage/"+url.PathEscape(key))
}

func (c *Client) LikeProject(ctx context.Context, projectID int64) error {
	_, err := c.Post(ctx, fmt.Sprintf("projects/%d/like", projectID), nil, nil)
	return err
}
func (c *Client) UnlikeProject(ctx context.Context, projectID int64) error {
	_, err := c.Post(ctx, fmt.Sprintf("projects/%d/unlike", projectID), nil, nil)
	return err
}
func (c *Client) ProjectFans(ctx context.Context, projectID int64, page, limit int) ([]User, Page, error) {
	query := url.Values{"page": {fmt.Sprint(page)}, "page_size": {fmt.Sprint(limit)}}
	var values []User
	header, err := c.Get(ctx, fmt.Sprintf("projects/%d/fans", projectID), query, &values)
	return values, pageFromHeaders(header, limit), err
}

func (c *Client) TransferProject(ctx context.Context, projectID int64, action string, body map[string]any) error {
	switch action {
	case "request", "start", "validate-token", "accept", "reject":
	default:
		return fmt.Errorf("invalid project transfer action %q", action)
	}
	_, err := c.Post(ctx, fmt.Sprintf("projects/%d/transfer_%s", projectID, strings.ReplaceAll(action, "-", "_")), body, nil)
	return err
}
