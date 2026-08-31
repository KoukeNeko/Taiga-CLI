package taiga

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ListWikiPages(ctx context.Context, projectID int64, pageNumber, limit int) ([]WikiPage, Page, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	if pageNumber > 0 {
		query.Set("page", fmt.Sprint(pageNumber))
	}
	if limit > 0 {
		query.Set("page_size", fmt.Sprint(limit))
	}
	var pages []WikiPage
	header, err := c.Get(ctx, "wiki", query, &pages)
	return pages, pageFromHeaders(header, limit), err
}

func (c *Client) GetWikiPageBySlug(ctx context.Context, projectID int64, slug string) (WikiPage, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}, "slug": []string{slug}}
	var page WikiPage
	_, err := c.Get(ctx, "wiki/by_slug", query, &page)
	return page, err
}

func (c *Client) CreateWikiPage(ctx context.Context, request CreateWikiPageRequest) (WikiPage, error) {
	var page WikiPage
	_, err := c.Post(ctx, "wiki", request, &page)
	return page, err
}

func (c *Client) UpdateWikiPage(ctx context.Context, id int64, request UpdateWikiPageRequest) (WikiPage, error) {
	var page WikiPage
	_, err := c.Patch(ctx, fmt.Sprintf("wiki/%d", id), request, &page)
	return page, err
}

func (c *Client) DeleteWikiPage(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("wiki/%d", id))
}
