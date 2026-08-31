package taiga

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) Search(ctx context.Context, projectID int64, text string) (SearchResponse, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}, "text": []string{text}}
	var response SearchResponse
	_, err := c.Get(ctx, "search", query, &response)
	return response, err
}
