package taiga

import (
	"context"
	"fmt"
	"net/url"
)

type TimelineEntry struct {
	ID              int64          `json:"id"`
	ContentType     int64          `json:"content_type"`
	ObjectID        int64          `json:"object_id"`
	Namespace       string         `json:"namespace"`
	EventType       string         `json:"event_type"`
	Project         *int64         `json:"project"`
	Data            map[string]any `json:"data"`
	DataContentType int64          `json:"data_content_type"`
	Created         string         `json:"created"`
}

func (c *Client) ProjectTimeline(ctx context.Context, projectID int64, onlyRelevant bool, pageNumber, limit int) ([]TimelineEntry, Page, error) {
	query := url.Values{}
	if onlyRelevant {
		query.Set("only_relevant", "true")
	}
	if pageNumber > 0 {
		query.Set("page", fmt.Sprint(pageNumber))
	}
	if limit > 0 {
		query.Set("page_size", fmt.Sprint(limit))
	}
	var entries []TimelineEntry
	header, err := c.Get(ctx, fmt.Sprintf("timeline/project/%d", projectID), query, &entries)
	return entries, pageFromHeaders(header, limit), err
}
