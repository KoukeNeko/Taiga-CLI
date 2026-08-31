package taiga

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ListEpics(ctx context.Context, projectID int64, pageNumber, limit int) ([]Epic, Page, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	if pageNumber > 0 {
		query.Set("page", fmt.Sprint(pageNumber))
	}
	if limit > 0 {
		query.Set("page_size", fmt.Sprint(limit))
	}
	var epics []Epic
	header, err := c.Get(ctx, "epics", query, &epics)
	return epics, pageFromHeaders(header, limit), err
}

func (c *Client) GetEpicByRef(ctx context.Context, projectSlug string, ref int) (Epic, error) {
	query := url.Values{"project__slug": []string{projectSlug}, "ref": []string{fmt.Sprint(ref)}}
	var epic Epic
	_, err := c.Get(ctx, "epics/by_ref", query, &epic)
	return epic, err
}

func (c *Client) CreateEpic(ctx context.Context, request CreateEpicRequest) (Epic, error) {
	var epic Epic
	_, err := c.Post(ctx, "epics", request, &epic)
	return epic, err
}

func (c *Client) UpdateEpic(ctx context.Context, id int64, request UpdateEpicRequest) (Epic, error) {
	var epic Epic
	_, err := c.Patch(ctx, fmt.Sprintf("epics/%d", id), request, &epic)
	return epic, err
}

func (c *Client) EpicStatuses(ctx context.Context, projectID int64) ([]EpicStatus, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	var statuses []EpicStatus
	_, err := c.Get(ctx, "epic-statuses", query, &statuses)
	return statuses, err
}

func (c *Client) ListEpicRelatedUserStories(ctx context.Context, epicID int64) ([]EpicRelatedUserStory, error) {
	var related []EpicRelatedUserStory
	_, err := c.Get(ctx, fmt.Sprintf("epics/%d/related_userstories", epicID), nil, &related)
	return related, err
}

func (c *Client) LinkEpicUserStory(ctx context.Context, epicID, storyID int64) (EpicRelatedUserStory, error) {
	request := CreateEpicRelatedUserStoryRequest{Epic: epicID, UserStory: storyID}
	var related EpicRelatedUserStory
	_, err := c.Post(ctx, fmt.Sprintf("epics/%d/related_userstories", epicID), request, &related)
	return related, err
}

func (c *Client) UnlinkEpicUserStory(ctx context.Context, epicID, storyID int64) error {
	return c.Delete(ctx, fmt.Sprintf("epics/%d/related_userstories/%d", epicID, storyID))
}
