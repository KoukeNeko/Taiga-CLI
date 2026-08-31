package taiga

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ListIssues(ctx context.Context, projectID int64, pageNumber, limit int) ([]Issue, Page, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	if pageNumber > 0 {
		query.Set("page", fmt.Sprint(pageNumber))
	}
	if limit > 0 {
		query.Set("page_size", fmt.Sprint(limit))
	}
	var issues []Issue
	header, err := c.Get(ctx, "issues", query, &issues)
	return issues, pageFromHeaders(header, limit), err
}

func (c *Client) GetIssueByRef(ctx context.Context, projectSlug string, ref int) (Issue, error) {
	query := url.Values{"project__slug": []string{projectSlug}, "ref": []string{fmt.Sprint(ref)}}
	var issue Issue
	_, err := c.Get(ctx, "issues/by_ref", query, &issue)
	return issue, err
}

func (c *Client) GetIssue(ctx context.Context, id int64) (Issue, error) {
	var issue Issue
	_, err := c.Get(ctx, fmt.Sprintf("issues/%d", id), nil, &issue)
	return issue, err
}

func (c *Client) CreateIssue(ctx context.Context, request CreateIssueRequest) (Issue, error) {
	var issue Issue
	_, err := c.Post(ctx, "issues", request, &issue)
	return issue, err
}

func (c *Client) UpdateIssue(ctx context.Context, id int64, request UpdateIssueRequest) (Issue, error) {
	var issue Issue
	_, err := c.Patch(ctx, fmt.Sprintf("issues/%d", id), request, &issue)
	return issue, err
}

func (c *Client) IssueStatuses(ctx context.Context, projectID int64) ([]IssueStatus, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	var values []IssueStatus
	_, err := c.Get(ctx, "issue-statuses", query, &values)
	return values, err
}

func (c *Client) IssuePriorities(ctx context.Context, projectID int64) ([]NamedMetadata, error) {
	return c.namedMetadata(ctx, "priorities", projectID)
}

func (c *Client) IssueSeverities(ctx context.Context, projectID int64) ([]NamedMetadata, error) {
	return c.namedMetadata(ctx, "severities", projectID)
}

func (c *Client) IssueTypes(ctx context.Context, projectID int64) ([]NamedMetadata, error) {
	return c.namedMetadata(ctx, "issue-types", projectID)
}

func (c *Client) namedMetadata(ctx context.Context, path string, projectID int64) ([]NamedMetadata, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	var values []NamedMetadata
	_, err := c.Get(ctx, path, query, &values)
	return values, err
}
