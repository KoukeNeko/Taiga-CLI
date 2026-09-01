package taiga

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

type Participant struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name,omitempty"`
}

type CommentVersion struct {
	Date        string         `json:"date"`
	Comment     string         `json:"comment"`
	CommentHTML string         `json:"comment_html,omitempty"`
	User        map[string]any `json:"user,omitempty"`
}

func (c *Client) DeleteWorkItem(ctx context.Context, resource string, id int64) error {
	path, err := workItemPath(resource)
	if err != nil {
		return err
	}
	operation := fmt.Sprintf("%s/%d", path, id)
	if err := c.Delete(ctx, operation); err != nil {
		return err
	}
	var result map[string]any
	if _, err := c.Get(ctx, operation, nil, &result); err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) && apiErr.Kind == KindNotFound {
			return nil
		}
		return &Error{Kind: KindAmbiguousCommit, Operation: "DELETE " + operation, Message: "work item was deleted but could not be verified", Retryable: false, Cause: err}
	}
	return &Error{Kind: KindAmbiguousCommit, Operation: "DELETE " + operation, Message: "Taiga still returns the work item after deletion", Retryable: false}
}

func (c *Client) ListParticipants(ctx context.Context, resource string, id int64, kind string, pageNumber, limit int) ([]Participant, Page, error) {
	path, err := workItemPath(resource)
	if err != nil {
		return nil, Page{}, err
	}
	if kind != "watchers" && kind != "voters" {
		return nil, Page{}, fmt.Errorf("participant kind must be watchers or voters")
	}
	if resource == "wiki" && kind == "voters" {
		return nil, Page{}, fmt.Errorf("wiki pages do not support voters")
	}
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("page", fmt.Sprint(pageNumber))
	}
	if limit > 0 {
		query.Set("page_size", fmt.Sprint(limit))
	}
	var participants []Participant
	header, err := c.Get(ctx, fmt.Sprintf("%s/%d/%s", path, id, kind), query, &participants)
	return participants, pageFromHeaders(header, limit), err
}

func (c *Client) FindHistoryEntry(ctx context.Context, resource string, id int64, historyID string) (HistoryEntry, error) {
	for pageNumber := 1; pageNumber <= 100; pageNumber++ {
		entries, page, err := c.History(ctx, resource, id, "all", pageNumber, 100)
		if err != nil {
			return HistoryEntry{}, err
		}
		for _, entry := range entries {
			if entry.ID == historyID {
				return entry, nil
			}
		}
		if page.Next == 0 || len(entries) == 0 {
			break
		}
	}
	return HistoryEntry{}, &Error{Kind: KindNotFound, Operation: "GET history", Message: fmt.Sprintf("history entry %q was not found", historyID), Retryable: false}
}

func (c *Client) EditComment(ctx context.Context, resource string, objectID int64, historyID, comment string) (HistoryEntry, error) {
	path, err := historyPath(resource)
	if err != nil {
		return HistoryEntry{}, err
	}
	operation := fmt.Sprintf("%s/%d/edit_comment", path, objectID)
	query := url.Values{"id": []string{historyID}}
	if _, err := c.PostQuery(ctx, operation, query, map[string]string{"comment": comment}, nil); err != nil {
		return HistoryEntry{}, err
	}
	entry, err := c.FindHistoryEntry(ctx, resource, objectID, historyID)
	if err != nil {
		return HistoryEntry{}, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + operation, Message: "comment may have been edited but could not be verified", Retryable: false, Cause: err}
	}
	if entry.Comment != comment || entry.EditCommentDate == "" {
		return entry, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + operation, Message: "Taiga did not confirm the edited comment", Retryable: false}
	}
	return entry, nil
}

func (c *Client) DeleteComment(ctx context.Context, resource string, objectID int64, historyID string) (HistoryEntry, error) {
	path, err := historyPath(resource)
	if err != nil {
		return HistoryEntry{}, err
	}
	operation := fmt.Sprintf("%s/%d/delete_comment", path, objectID)
	query := url.Values{"id": []string{historyID}}
	if _, err := c.PostQuery(ctx, operation, query, nil, nil); err != nil {
		return HistoryEntry{}, err
	}
	entry, err := c.FindHistoryEntry(ctx, resource, objectID, historyID)
	if err != nil {
		return HistoryEntry{}, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + operation, Message: "comment may have been deleted but could not be verified", Retryable: false, Cause: err}
	}
	if entry.DeleteCommentDate == "" {
		return entry, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + operation, Message: "Taiga did not confirm the deleted comment", Retryable: false}
	}
	return entry, nil
}

func (c *Client) UndeleteComment(ctx context.Context, resource string, objectID int64, historyID string) (HistoryEntry, error) {
	path, err := historyPath(resource)
	if err != nil {
		return HistoryEntry{}, err
	}
	operation := fmt.Sprintf("%s/%d/undelete_comment", path, objectID)
	query := url.Values{"id": []string{historyID}}
	if _, err := c.PostQuery(ctx, operation, query, nil, nil); err != nil {
		return HistoryEntry{}, err
	}
	entry, err := c.FindHistoryEntry(ctx, resource, objectID, historyID)
	if err != nil {
		return HistoryEntry{}, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + operation, Message: "comment may have been restored but could not be verified", Retryable: false, Cause: err}
	}
	if entry.DeleteCommentDate != "" {
		return entry, &Error{Kind: KindAmbiguousCommit, Operation: "POST " + operation, Message: "Taiga did not confirm the restored comment", Retryable: false}
	}
	return entry, nil
}

func (c *Client) CommentVersions(ctx context.Context, resource string, objectID int64, historyID string) ([]CommentVersion, error) {
	path, err := historyPath(resource)
	if err != nil {
		return nil, err
	}
	query := url.Values{"id": []string{historyID}}
	var versions []CommentVersion
	_, err = c.Get(ctx, fmt.Sprintf("%s/%d/comment_versions", path, objectID), query, &versions)
	return versions, err
}
