package taiga

import (
	"context"
	"fmt"
	"net/url"
)

type WikiLink struct {
	ID      int64  `json:"id"`
	Project int64  `json:"project"`
	Title   string `json:"title"`
	Href    string `json:"href"`
	Order   int64  `json:"order"`
}

type CreateWikiLinkRequest struct {
	Project int64  `json:"project"`
	Title   string `json:"title"`
	Order   *int64 `json:"order,omitempty"`
}

type UpdateWikiLinkRequest struct {
	Title *string `json:"title,omitempty"`
	Order *int64  `json:"order,omitempty"`
}

func (c *Client) ListWikiLinks(ctx context.Context, projectID int64, pageNumber, limit int) ([]WikiLink, Page, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	if pageNumber > 0 {
		query.Set("page", fmt.Sprint(pageNumber))
	}
	if limit > 0 {
		query.Set("page_size", fmt.Sprint(limit))
	}
	var links []WikiLink
	header, err := c.Get(ctx, "wiki-links", query, &links)
	return links, pageFromHeaders(header, limit), err
}

func (c *Client) GetWikiLink(ctx context.Context, id int64) (WikiLink, error) {
	var link WikiLink
	_, err := c.Get(ctx, fmt.Sprintf("wiki-links/%d", id), nil, &link)
	return link, err
}

func (c *Client) CreateWikiLink(ctx context.Context, request CreateWikiLinkRequest) (WikiLink, error) {
	var link WikiLink
	_, err := c.Post(ctx, "wiki-links", request, &link)
	return link, err
}

func (c *Client) UpdateWikiLink(ctx context.Context, id int64, request UpdateWikiLinkRequest) (WikiLink, error) {
	var link WikiLink
	_, err := c.Patch(ctx, fmt.Sprintf("wiki-links/%d", id), request, &link)
	return link, err
}

func (c *Client) DeleteWikiLink(ctx context.Context, id int64) error {
	return c.deleteAndVerify(ctx, "wiki link", "wiki-links", id, nil)
}
