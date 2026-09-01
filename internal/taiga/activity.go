package taiga

import (
	"context"
	"fmt"
	"net/url"
)

func workItemPath(resource string) (string, error) {
	switch resource {
	case "issue":
		return "issues", nil
	case "story":
		return "userstories", nil
	case "task":
		return "tasks", nil
	case "wiki":
		return "wiki", nil
	case "epic":
		return "epics", nil
	default:
		return "", fmt.Errorf("unsupported work item resource %q", resource)
	}
}

func historyPath(resource string) (string, error) {
	switch resource {
	case "issue":
		return "history/issue", nil
	case "story":
		return "history/userstory", nil
	case "task":
		return "history/task", nil
	case "wiki":
		return "history/wiki", nil
	case "epic":
		return "history/epic", nil
	default:
		return "", fmt.Errorf("unsupported history resource %q", resource)
	}
}

func (c *Client) SetWatching(ctx context.Context, resource string, id int64, watching bool) (bool, error) {
	path, err := workItemPath(resource)
	if err != nil {
		return false, err
	}
	action := "unwatch"
	if watching {
		action = "watch"
	}
	operation := fmt.Sprintf("%s/%d/%s", path, id, action)
	if _, err := c.Post(ctx, operation, nil, nil); err != nil {
		return false, err
	}
	verified, err := c.Watching(ctx, resource, id)
	if err != nil {
		return false, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + operation, Message: "watch state changed but could not be verified", Retryable: false, Cause: err}
	}
	if verified != watching {
		return verified, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + operation, Message: "Taiga did not confirm the requested watch state", Retryable: false}
	}
	return verified, nil
}

func (c *Client) Watching(ctx context.Context, resource string, id int64) (bool, error) {
	path, err := workItemPath(resource)
	if err != nil {
		return false, err
	}
	var result struct {
		IsWatcher bool `json:"is_watcher"`
	}
	_, err = c.Get(ctx, fmt.Sprintf("%s/%d", path, id), nil, &result)
	return result.IsWatcher, err
}

func (c *Client) SetVoting(ctx context.Context, resource string, id int64, voting bool) (bool, error) {
	path, err := workItemPath(resource)
	if err != nil {
		return false, err
	}
	action := "downvote"
	if voting {
		action = "upvote"
	}
	operation := fmt.Sprintf("%s/%d/%s", path, id, action)
	if _, err := c.Post(ctx, operation, nil, nil); err != nil {
		return false, err
	}
	verified, err := c.Voting(ctx, resource, id)
	if err != nil {
		return false, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + operation, Message: "vote state changed but could not be verified", Retryable: false, Cause: err}
	}
	if verified != voting {
		return verified, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + operation, Message: "Taiga did not confirm the requested vote state", Retryable: false}
	}
	return verified, nil
}

func (c *Client) Voting(ctx context.Context, resource string, id int64) (bool, error) {
	path, err := workItemPath(resource)
	if err != nil {
		return false, err
	}
	var result struct {
		IsVoter bool `json:"is_voter"`
	}
	_, err = c.Get(ctx, fmt.Sprintf("%s/%d", path, id), nil, &result)
	return result.IsVoter, err
}

func (c *Client) History(ctx context.Context, resource string, id int64, historyType string, pageNumber, limit int) ([]HistoryEntry, Page, error) {
	path, err := historyPath(resource)
	if err != nil {
		return nil, Page{}, err
	}
	query := url.Values{}
	if historyType != "" && historyType != "all" {
		query.Set("type", historyType)
	}
	if pageNumber > 0 {
		query.Set("page", fmt.Sprint(pageNumber))
	}
	if limit > 0 {
		query.Set("page_size", fmt.Sprint(limit))
	}
	var entries []HistoryEntry
	header, err := c.Get(ctx, fmt.Sprintf("%s/%d", path, id), query, &entries)
	return entries, pageFromHeaders(header, limit), err
}
