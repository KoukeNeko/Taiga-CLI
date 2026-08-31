package taiga

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ListUserStories(ctx context.Context, projectID int64, pageNumber, limit int, milestone *int64, backlog bool, orderBy string) ([]UserStory, Page, error) {
	if orderBy == "" {
		orderBy = "backlog_order"
	}
	query := url.Values{"project": []string{fmt.Sprint(projectID)}, "order_by": []string{orderBy}}
	if pageNumber > 0 {
		query.Set("page", fmt.Sprint(pageNumber))
	}
	if limit > 0 {
		query.Set("page_size", fmt.Sprint(limit))
	}
	if milestone != nil {
		query.Set("milestone", fmt.Sprint(*milestone))
	} else if backlog {
		query.Set("milestone", "null")
	}
	var stories []UserStory
	header, err := c.Get(ctx, "userstories", query, &stories)
	return stories, pageFromHeaders(header, limit), err
}

func (c *Client) GetUserStoryByRef(ctx context.Context, projectSlug string, ref int) (UserStory, error) {
	query := url.Values{"project__slug": []string{projectSlug}, "ref": []string{fmt.Sprint(ref)}}
	var story UserStory
	_, err := c.Get(ctx, "userstories/by_ref", query, &story)
	return story, err
}

func (c *Client) CreateUserStory(ctx context.Context, request CreateUserStoryRequest) (UserStory, error) {
	var story UserStory
	_, err := c.Post(ctx, "userstories", request, &story)
	return story, err
}

func (c *Client) UpdateUserStory(ctx context.Context, id int64, request UpdateUserStoryRequest) (UserStory, error) {
	var story UserStory
	_, err := c.Patch(ctx, fmt.Sprintf("userstories/%d", id), request, &story)
	return story, err
}

func (c *Client) UserStoryStatuses(ctx context.Context, projectID int64) ([]UserStoryStatus, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	var values []UserStoryStatus
	_, err := c.Get(ctx, "userstory-statuses", query, &values)
	return values, err
}

func (c *Client) Milestones(ctx context.Context, projectID int64, closed *bool) ([]Milestone, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	if closed != nil {
		query.Set("closed", fmt.Sprint(*closed))
	}
	var values []Milestone
	_, err := c.Get(ctx, "milestones", query, &values)
	return values, err
}
