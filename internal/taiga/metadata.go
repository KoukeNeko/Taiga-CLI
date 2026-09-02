package taiga

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type WorkflowMetadata struct {
	ID         int64    `json:"id"`
	Project    int64    `json:"project"`
	Name       string   `json:"name"`
	Slug       string   `json:"slug,omitempty"`
	Order      int      `json:"order"`
	Color      string   `json:"color,omitempty"`
	IsClosed   bool     `json:"is_closed,omitempty"`
	IsArchived bool     `json:"is_archived,omitempty"`
	WIPLimit   *int     `json:"wip_limit,omitempty"`
	Value      *float64 `json:"value,omitempty"`
}

func MetadataPath(kind string) (string, error) {
	switch kind {
	case "epic-status":
		return "epic-statuses", nil
	case "story-status":
		return "userstory-statuses", nil
	case "task-status":
		return "task-statuses", nil
	case "issue-status":
		return "issue-statuses", nil
	case "points":
		return "points", nil
	case "priority":
		return "priorities", nil
	case "severity":
		return "severities", nil
	case "issue-type":
		return "issue-types", nil
	default:
		return "", fmt.Errorf("unsupported workflow metadata kind %q", kind)
	}
}

func NormalizeMetadataKind(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "epic-status", "epic-statuses":
		return "epic-status", nil
	case "story-status", "story-statuses", "userstory-status", "userstory-statuses", "us-status":
		return "story-status", nil
	case "task-status", "task-statuses":
		return "task-status", nil
	case "issue-status", "issue-statuses":
		return "issue-status", nil
	case "point", "points":
		return "points", nil
	case "priority", "priorities":
		return "priority", nil
	case "severity", "severities":
		return "severity", nil
	case "issue-type", "issue-types", "type", "types":
		return "issue-type", nil
	default:
		return "", fmt.Errorf("metadata kind must be epic-status, story-status, task-status, issue-status, points, priority, severity, or issue-type")
	}
}

func (c *Client) ListWorkflowMetadata(ctx context.Context, kind string, projectID int64) ([]WorkflowMetadata, error) {
	path, err := MetadataPath(kind)
	if err != nil {
		return nil, err
	}
	query := url.Values{"project": []string{fmt.Sprint(projectID)}, "page_size": []string{"1000"}}
	var values []WorkflowMetadata
	_, err = c.Get(ctx, path, query, &values)
	return values, err
}

func (c *Client) GetWorkflowMetadata(ctx context.Context, kind string, id int64) (WorkflowMetadata, error) {
	path, err := MetadataPath(kind)
	if err != nil {
		return WorkflowMetadata{}, err
	}
	var value WorkflowMetadata
	_, err = c.Get(ctx, fmt.Sprintf("%s/%d", path, id), nil, &value)
	return value, err
}

func (c *Client) CreateWorkflowMetadata(ctx context.Context, kind string, projectID int64, fields map[string]any) (WorkflowMetadata, error) {
	path, err := MetadataPath(kind)
	if err != nil {
		return WorkflowMetadata{}, err
	}
	body := map[string]any{"project": projectID}
	for key, value := range fields {
		body[key] = value
	}
	var created WorkflowMetadata
	_, err = c.Post(ctx, path, body, &created)
	return created, err
}

func (c *Client) UpdateWorkflowMetadata(ctx context.Context, kind string, id int64, fields map[string]any) (WorkflowMetadata, error) {
	path, err := MetadataPath(kind)
	if err != nil {
		return WorkflowMetadata{}, err
	}
	var updated WorkflowMetadata
	_, err = c.Patch(ctx, fmt.Sprintf("%s/%d", path, id), fields, &updated)
	return updated, err
}

func (c *Client) DeleteWorkflowMetadata(ctx context.Context, kind string, id, moveTo int64) error {
	path, err := MetadataPath(kind)
	if err != nil {
		return err
	}
	operation := fmt.Sprintf("%s/%d", path, id)
	query := url.Values{"moveTo": []string{fmt.Sprint(moveTo)}}
	if _, err := c.doJSON(ctx, http.MethodDelete, operation, query, nil, nil, false, mayCommit); err != nil {
		return err
	}
	if _, err := c.GetWorkflowMetadata(ctx, kind, id); err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) && apiErr.Kind == KindNotFound {
			return nil
		}
		return &Error{Kind: KindAmbiguousCommit, Operation: "DELETE " + operation, Message: "metadata was deleted but could not be verified", Retryable: false, Cause: err}
	}
	return &Error{Kind: KindAmbiguousCommit, Operation: "DELETE " + operation, Message: "Taiga still returns the metadata after deletion", Retryable: false}
}
