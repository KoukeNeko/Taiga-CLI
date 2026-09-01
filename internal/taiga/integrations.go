package taiga

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

func NormalizeImporterProvider(provider string) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case "github", "gitlab", "jira", "trello", "asana", "pivotal":
		return provider, nil
	default:
		return "", fmt.Errorf("provider must be github, gitlab, jira, trello, asana, or pivotal")
	}
}

func (c *Client) ImporterCall(ctx context.Context, provider, action string, fields map[string]any) (any, error) {
	provider, err := NormalizeImporterProvider(provider)
	if err != nil {
		return nil, err
	}
	path := "importers/" + provider + "/" + strings.ReplaceAll(action, "-", "_")
	var result any
	if action == "auth-url" {
		query := url.Values{}
		for key, value := range fields {
			query.Set(key, fmt.Sprint(value))
		}
		_, err = c.Get(ctx, path, query, &result)
	} else {
		_, err = c.Post(ctx, path, fields, &result)
	}
	return result, err
}

func (c *Client) BulkMoveToMilestone(ctx context.Context, resource string, projectID, milestoneID int64, ids []int64) error {
	body := map[string]any{"project_id": projectID, "milestone_id": milestoneID}
	path := ""
	switch resource {
	case "story":
		items := make([]map[string]any, len(ids))
		for i, id := range ids {
			items[i] = map[string]any{"us_id": id, "order": i + 1}
		}
		body["bulk_stories"], path = items, "userstories/bulk_update_milestone"
	case "task":
		items := make([]map[string]any, len(ids))
		for i, id := range ids {
			items[i] = map[string]any{"task_id": id, "order": i + 1}
		}
		body["bulk_tasks"], path = items, "tasks/bulk_update_milestone"
	case "issue":
		items := make([]map[string]any, len(ids))
		for i, id := range ids {
			items[i] = map[string]any{"issue_id": id}
		}
		body["bulk_issues"], path = items, "issues/bulk_update_milestone"
	default:
		return fmt.Errorf("batch move resource must be story, task, or issue")
	}
	_, err := c.Post(ctx, path, body, nil)
	return err
}

func (c *Client) BulkOrderStories(ctx context.Context, projectID int64, view string, ids []int64, options map[string]any) (any, error) {
	body := map[string]any{"project_id": projectID, "bulk_userstories": ids}
	for key, value := range options {
		body[key] = value
	}
	var result any
	_, err := c.Post(ctx, "userstories/bulk_update_"+view+"_order", body, &result)
	return result, err
}

func (c *Client) BulkOrderTasks(ctx context.Context, projectID int64, view string, orders map[int64]int, options map[string]any) (any, error) {
	items := make([]map[string]any, 0, len(orders))
	ids := make([]int64, 0, len(orders))
	for id := range orders {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		order := orders[id]
		items = append(items, map[string]any{"task_id": id, "order": order})
	}
	body := map[string]any{"project_id": projectID, "bulk_tasks": items}
	for key, value := range options {
		body[key] = value
	}
	var result any
	_, err := c.Post(ctx, "tasks/bulk_update_"+view+"_order", body, &result)
	return result, err
}
