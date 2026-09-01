package taiga

import (
	"context"
	"fmt"
)

type BulkCreateRequest struct {
	ProjectID   int64
	Subjects    string
	StatusID    *int64
	MilestoneID *int64
	StoryID     *int64
}

type BulkCreatedItem struct {
	ID                 int64          `json:"id"`
	Ref                int            `json:"ref"`
	Project            int64          `json:"project"`
	Subject            string         `json:"subject"`
	Version            int            `json:"version"`
	Status             int64          `json:"status"`
	StatusExtraInfo    ExtraInfo      `json:"status_extra_info"`
	Milestone          *int64         `json:"milestone"`
	MilestoneSlug      string         `json:"milestone_slug,omitempty"`
	UserStory          *int64         `json:"user_story"`
	UserStoryExtraInfo *TaskStoryInfo `json:"user_story_extra_info,omitempty"`
}

func (c *Client) BulkCreate(ctx context.Context, resource string, request BulkCreateRequest) ([]BulkCreatedItem, error) {
	path := ""
	body := map[string]any{"project_id": request.ProjectID}
	switch resource {
	case "epic":
		path = "epics/bulk_create"
		body["bulk_epics"] = request.Subjects
	case "story":
		path = "userstories/bulk_create"
		body["bulk_stories"] = request.Subjects
	case "issue":
		path = "issues/bulk_create"
		body["bulk_issues"] = request.Subjects
		body["milestone_id"] = request.MilestoneID
	case "task":
		path = "tasks/bulk_create"
		body["bulk_tasks"] = request.Subjects
		if request.MilestoneID == nil {
			return nil, fmt.Errorf("task bulk creation requires a milestone")
		}
		body["milestone_id"] = *request.MilestoneID
		if request.StoryID != nil {
			body["us_id"] = *request.StoryID
		}
	default:
		return nil, fmt.Errorf("unsupported bulk resource %q", resource)
	}
	if request.StatusID != nil {
		body["status_id"] = *request.StatusID
	}
	var created []BulkCreatedItem
	_, err := c.Post(ctx, path, body, &created)
	return created, err
}
