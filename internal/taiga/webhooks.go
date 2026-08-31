package taiga

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ListWebhooks(ctx context.Context, projectID int64) ([]Webhook, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}, "page_size": []string{"1000"}}
	var hooks []Webhook
	_, err := c.Get(ctx, "webhooks", query, &hooks)
	return hooks, err
}

func (c *Client) GetWebhook(ctx context.Context, id int64) (Webhook, error) {
	var hook Webhook
	_, err := c.Get(ctx, fmt.Sprintf("webhooks/%d", id), nil, &hook)
	return hook, err
}

func (c *Client) CreateWebhook(ctx context.Context, request CreateWebhookRequest) (Webhook, error) {
	var hook Webhook
	_, err := c.Post(ctx, "webhooks", request, &hook)
	return hook, err
}

func (c *Client) UpdateWebhook(ctx context.Context, id int64, request UpdateWebhookRequest) (Webhook, error) {
	var hook Webhook
	_, err := c.Patch(ctx, fmt.Sprintf("webhooks/%d", id), request, &hook)
	return hook, err
}

func (c *Client) TestWebhook(ctx context.Context, id int64) (WebhookLog, error) {
	var log WebhookLog
	_, err := c.Post(ctx, fmt.Sprintf("webhooks/%d/test", id), nil, &log)
	return log, err
}

func (c *Client) DeleteWebhook(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("webhooks/%d", id))
}
