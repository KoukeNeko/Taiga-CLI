package taiga

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type DueDate struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Order     int    `json:"order"`
	ByDefault bool   `json:"by_default"`
	DaysToDue int    `json:"days_to_due"`
	Color     string `json:"color"`
	Project   int64  `json:"project"`
}

func dueDatePath(resource string) (string, error) {
	switch resource {
	case "story":
		return "userstory-due-dates", nil
	case "task":
		return "task-due-dates", nil
	case "issue":
		return "issue-due-dates", nil
	default:
		return "", fmt.Errorf("due-date resource must be story, task, or issue")
	}
}

func (c *Client) ListDueDates(ctx context.Context, resource string, projectID int64) ([]DueDate, error) {
	path, err := dueDatePath(resource)
	if err != nil {
		return nil, err
	}
	var values []DueDate
	_, err = c.Get(ctx, path, url.Values{"project": {fmt.Sprint(projectID)}, "page_size": {"1000"}}, &values)
	return values, err
}

func (c *Client) GetDueDate(ctx context.Context, resource string, id int64) (DueDate, error) {
	path, err := dueDatePath(resource)
	if err != nil {
		return DueDate{}, err
	}
	var value DueDate
	_, err = c.Get(ctx, fmt.Sprintf("%s/%d", path, id), nil, &value)
	return value, err
}

func (c *Client) CreateDueDate(ctx context.Context, resource string, projectID int64, fields map[string]any) (DueDate, error) {
	path, err := dueDatePath(resource)
	if err != nil {
		return DueDate{}, err
	}
	body := map[string]any{"project": projectID}
	for key, value := range fields {
		body[key] = value
	}
	var created DueDate
	_, err = c.Post(ctx, path, body, &created)
	return created, err
}

func (c *Client) UpdateDueDate(ctx context.Context, resource string, id int64, fields map[string]any) (DueDate, error) {
	path, err := dueDatePath(resource)
	if err != nil {
		return DueDate{}, err
	}
	var updated DueDate
	_, err = c.Patch(ctx, fmt.Sprintf("%s/%d", path, id), fields, &updated)
	return updated, err
}

func (c *Client) DeleteDueDate(ctx context.Context, resource string, id int64) error {
	path, err := dueDatePath(resource)
	if err != nil {
		return err
	}
	return c.deleteAndVerify(ctx, path, id, nil)
}

type SwimlaneStatus struct {
	ID                        int64  `json:"id"`
	Name                      string `json:"name"`
	Slug                      string `json:"slug"`
	Order                     int    `json:"order"`
	WIPLimit                  *int   `json:"wip_limit"`
	SwimlaneUserStoryStatusID int64  `json:"swimlane_userstory_status_id"`
}

type Swimlane struct {
	ID       int64            `json:"id"`
	Name     string           `json:"name"`
	Order    int              `json:"order"`
	Project  int64            `json:"project"`
	Statuses []SwimlaneStatus `json:"statuses,omitempty"`
}

type SwimlaneWIP struct {
	ID       int64 `json:"id"`
	Status   int64 `json:"status"`
	Swimlane int64 `json:"swimlane"`
	WIPLimit *int  `json:"wip_limit"`
}

func (c *Client) ListSwimlanes(ctx context.Context, projectID int64) ([]Swimlane, error) {
	var values []Swimlane
	_, err := c.Get(ctx, "swimlanes", url.Values{"project": {fmt.Sprint(projectID)}, "page_size": {"1000"}}, &values)
	return values, err
}

func (c *Client) GetSwimlane(ctx context.Context, id int64) (Swimlane, error) {
	var value Swimlane
	_, err := c.Get(ctx, fmt.Sprintf("swimlanes/%d", id), nil, &value)
	return value, err
}

func (c *Client) CreateSwimlane(ctx context.Context, projectID int64, name string, order int) (Swimlane, error) {
	var value Swimlane
	_, err := c.Post(ctx, "swimlanes", map[string]any{"project": projectID, "name": name, "order": order}, &value)
	return value, err
}

func (c *Client) UpdateSwimlane(ctx context.Context, id int64, fields map[string]any) (Swimlane, error) {
	var value Swimlane
	_, err := c.Patch(ctx, fmt.Sprintf("swimlanes/%d", id), fields, &value)
	return value, err
}

func (c *Client) DeleteSwimlane(ctx context.Context, id int64, moveTo *int64) error {
	var query url.Values
	if moveTo != nil {
		query = url.Values{"moveTo": {fmt.Sprint(*moveTo)}}
	}
	return c.deleteAndVerify(ctx, "swimlanes", id, query)
}

func (c *Client) ListSwimlaneWIP(ctx context.Context, projectID int64) ([]SwimlaneWIP, error) {
	var values []SwimlaneWIP
	_, err := c.Get(ctx, "swimlane-userstory-statuses", url.Values{"project": {fmt.Sprint(projectID)}, "page_size": {"1000"}}, &values)
	return values, err
}

func (c *Client) UpdateSwimlaneWIP(ctx context.Context, id int64, limit *int) (SwimlaneWIP, error) {
	var value SwimlaneWIP
	_, err := c.Patch(ctx, fmt.Sprintf("swimlane-userstory-statuses/%d", id), map[string]any{"wip_limit": limit}, &value)
	return value, err
}

func (c *Client) ProjectTags(ctx context.Context, projectID int64) (map[string]*string, error) {
	values := map[string]*string{}
	_, err := c.Get(ctx, fmt.Sprintf("projects/%d/tags_colors", projectID), nil, &values)
	return values, err
}

func (c *Client) CreateProjectTag(ctx context.Context, projectID int64, tag, color string) error {
	body := map[string]any{"tag": tag}
	if color != "" {
		body["color"] = color
	}
	_, err := c.Post(ctx, fmt.Sprintf("projects/%d/create_tag", projectID), body, nil)
	return err
}

func (c *Client) EditProjectTag(ctx context.Context, projectID int64, from, to, color string, colorSet bool) error {
	body := map[string]any{"from_tag": from}
	if to != "" {
		body["to_tag"] = to
	}
	if colorSet {
		body["color"] = color
	}
	_, err := c.Post(ctx, fmt.Sprintf("projects/%d/edit_tag", projectID), body, nil)
	return err
}

func (c *Client) DeleteProjectTag(ctx context.Context, projectID int64, tag string) error {
	_, err := c.Post(ctx, fmt.Sprintf("projects/%d/delete_tag", projectID), map[string]any{"tag": tag}, nil)
	return err
}

func (c *Client) MixProjectTags(ctx context.Context, projectID int64, from []string, to string) error {
	_, err := c.Post(ctx, fmt.Sprintf("projects/%d/mix_tags", projectID), map[string]any{"from_tags": from, "to_tag": to}, nil)
	return err
}

func (c *Client) deleteAndVerify(ctx context.Context, path string, id int64, query url.Values) error {
	operation := fmt.Sprintf("%s/%d", path, id)
	if _, err := c.doJSON(ctx, http.MethodDelete, operation, query, nil, nil, false, mayCommit); err != nil {
		return err
	}
	var ignored any
	if _, err := c.Get(ctx, operation, nil, &ignored); err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) && apiErr.Kind == KindNotFound {
			return nil
		}
		return &Error{Kind: KindAmbiguousCommit, Operation: "DELETE " + operation, Message: "resource was deleted but could not be verified", Cause: err}
	}
	return &Error{Kind: KindAmbiguousCommit, Operation: "DELETE " + operation, Message: "Taiga still returns the resource after deletion"}
}
