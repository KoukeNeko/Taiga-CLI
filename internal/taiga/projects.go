package taiga

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ListProjects(ctx context.Context, pageNumber, limit int) ([]Project, Page, error) {
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("page", fmt.Sprint(pageNumber))
	}
	if limit > 0 {
		query.Set("page_size", fmt.Sprint(limit))
	}
	var projects []Project
	header, err := c.Get(ctx, "projects", query, &projects)
	return projects, pageFromHeaders(header, limit), err
}

func (c *Client) GetProjectBySlug(ctx context.Context, slug string) (Project, error) {
	query := url.Values{"slug": []string{slug}}
	var project Project
	_, err := c.Get(ctx, "projects/by_slug", query, &project)
	return project, err
}

func (c *Client) GetProject(ctx context.Context, id int64) (Project, error) {
	var project Project
	_, err := c.Get(ctx, fmt.Sprintf("projects/%d", id), nil, &project)
	return project, err
}

func (c *Client) ListProjectUsers(ctx context.Context, projectID int64) ([]User, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	var users []User
	_, err := c.Get(ctx, "users", query, &users)
	return users, err
}

func (c *Client) ListProjectTemplates(ctx context.Context) ([]ProjectTemplate, error) {
	var templates []ProjectTemplate
	_, err := c.Get(ctx, "project-templates", nil, &templates)
	return templates, err
}

func (c *Client) CreateProject(ctx context.Context, request CreateProjectRequest) (Project, error) {
	var project Project
	_, err := c.Post(ctx, "projects", request, &project)
	return project, err
}

func (c *Client) UpdateProject(ctx context.Context, id int64, request UpdateProjectRequest) (Project, error) {
	var project Project
	_, err := c.Patch(ctx, fmt.Sprintf("projects/%d", id), request, &project)
	return project, err
}

func (c *Client) DeleteProject(ctx context.Context, id int64) error {
	return c.Delete(ctx, fmt.Sprintf("projects/%d", id))
}
