package taiga

import (
	"context"
	"fmt"
	"net/url"
)

func customFieldPaths(resource string) (string, string, error) {
	switch resource {
	case "epic":
		return "epic-custom-attributes", "epics/custom-attributes-values", nil
	case "story":
		return "userstory-custom-attributes", "userstories/custom-attributes-values", nil
	case "task":
		return "task-custom-attributes", "tasks/custom-attributes-values", nil
	case "issue":
		return "issue-custom-attributes", "issues/custom-attributes-values", nil
	default:
		return "", "", fmt.Errorf("custom field resource must be epic, story, task, or issue")
	}
}

func (c *Client) ListCustomFields(ctx context.Context, resource string, projectID int64) ([]CustomField, error) {
	path, _, err := customFieldPaths(resource)
	if err != nil {
		return nil, err
	}
	query := url.Values{"project": []string{fmt.Sprint(projectID)}, "page_size": []string{"1000"}}
	var fields []CustomField
	_, err = c.Get(ctx, path, query, &fields)
	return fields, err
}

func (c *Client) CreateCustomField(ctx context.Context, resource string, request CreateCustomFieldRequest) (CustomField, error) {
	path, _, err := customFieldPaths(resource)
	if err != nil {
		return CustomField{}, err
	}
	var field CustomField
	_, err = c.Post(ctx, path, request, &field)
	return field, err
}

func (c *Client) UpdateCustomField(ctx context.Context, resource string, id int64, request UpdateCustomFieldRequest) (CustomField, error) {
	path, _, err := customFieldPaths(resource)
	if err != nil {
		return CustomField{}, err
	}
	var field CustomField
	_, err = c.Patch(ctx, fmt.Sprintf("%s/%d", path, id), request, &field)
	return field, err
}

func (c *Client) DeleteCustomField(ctx context.Context, resource string, id int64) error {
	path, _, err := customFieldPaths(resource)
	if err != nil {
		return err
	}
	return c.Delete(ctx, fmt.Sprintf("%s/%d", path, id))
}

func (c *Client) GetCustomFieldValues(ctx context.Context, resource string, objectID int64) (CustomFieldValues, error) {
	_, path, err := customFieldPaths(resource)
	if err != nil {
		return CustomFieldValues{}, err
	}
	var values CustomFieldValues
	_, err = c.Get(ctx, fmt.Sprintf("%s/%d", path, objectID), nil, &values)
	values.Resource = objectID
	return values, err
}

func (c *Client) UpdateCustomFieldValues(ctx context.Context, resource string, objectID int64, request UpdateCustomFieldValuesRequest) (CustomFieldValues, error) {
	_, path, err := customFieldPaths(resource)
	if err != nil {
		return CustomFieldValues{}, err
	}
	var values CustomFieldValues
	_, err = c.Patch(ctx, fmt.Sprintf("%s/%d", path, objectID), request, &values)
	values.Resource = objectID
	return values, err
}
