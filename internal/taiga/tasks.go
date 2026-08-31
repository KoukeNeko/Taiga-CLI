package taiga

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ListTasks(ctx context.Context, projectID int64, storyID *int64, pageNumber, limit int, orderBy string) ([]Task, Page, error) {
	if orderBy == "" {
		orderBy = "us_order"
	}
	query := url.Values{"project": []string{fmt.Sprint(projectID)}, "order_by": []string{orderBy}}
	if storyID != nil {
		query.Set("user_story", fmt.Sprint(*storyID))
	}
	if pageNumber > 0 {
		query.Set("page", fmt.Sprint(pageNumber))
	}
	if limit > 0 {
		query.Set("page_size", fmt.Sprint(limit))
	}
	var tasks []Task
	header, err := c.Get(ctx, "tasks", query, &tasks)
	return tasks, pageFromHeaders(header, limit), err
}

func (c *Client) GetTaskByRef(ctx context.Context, projectSlug string, ref int) (Task, error) {
	query := url.Values{"project__slug": []string{projectSlug}, "ref": []string{fmt.Sprint(ref)}}
	var task Task
	_, err := c.Get(ctx, "tasks/by_ref", query, &task)
	return task, err
}

func (c *Client) CreateTask(ctx context.Context, request CreateTaskRequest) (Task, error) {
	var task Task
	_, err := c.Post(ctx, "tasks", request, &task)
	return task, err
}

func (c *Client) UpdateTask(ctx context.Context, id int64, request UpdateTaskRequest) (Task, error) {
	var task Task
	_, err := c.Patch(ctx, fmt.Sprintf("tasks/%d", id), request, &task)
	return task, err
}

func (c *Client) TaskStatuses(ctx context.Context, projectID int64) ([]TaskStatus, error) {
	query := url.Values{"project": []string{fmt.Sprint(projectID)}}
	var values []TaskStatus
	_, err := c.Get(ctx, "task-statuses", query, &values)
	return values, err
}
