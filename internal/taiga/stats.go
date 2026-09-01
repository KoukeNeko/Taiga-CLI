package taiga

import (
	"context"
	"fmt"
)

type ProjectMilestoneStats struct {
	Name            string   `json:"name"`
	Optimal         float64  `json:"optimal"`
	Evolution       *float64 `json:"evolution"`
	TeamIncrement   float64  `json:"team-increment"`
	ClientIncrement float64  `json:"client-increment"`
}

type ProjectStats struct {
	Name                  string                  `json:"name"`
	TotalMilestones       *int                    `json:"total_milestones"`
	TotalPoints           *float64                `json:"total_points"`
	ClosedPoints          float64                 `json:"closed_points"`
	ClosedPointsPerRole   map[string]float64      `json:"closed_points_per_role"`
	DefinedPoints         float64                 `json:"defined_points"`
	DefinedPointsPerRole  map[string]float64      `json:"defined_points_per_role"`
	AssignedPoints        float64                 `json:"assigned_points"`
	AssignedPointsPerRole map[string]float64      `json:"assigned_points_per_role"`
	Milestones            []ProjectMilestoneStats `json:"milestones"`
	Speed                 float64                 `json:"speed"`
}

type StatsCount struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
	Color    string `json:"color,omitempty"`
	Count    int    `json:"count"`
}

type StatsSeries struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
	Data  []int  `json:"data"`
}

type OpenClosedSeries struct {
	Open   []int `json:"open"`
	Closed []int `json:"closed"`
}

type LastFourWeeksStats struct {
	ByOpenClosed OpenClosedSeries       `json:"by_open_closed"`
	BySeverity   map[string]StatsSeries `json:"by_severity"`
	ByPriority   map[string]StatsSeries `json:"by_priority"`
	ByStatus     map[string]StatsSeries `json:"by_status"`
}

type ProjectIssueStats struct {
	TotalIssues         int                   `json:"total_issues"`
	OpenedIssues        int                   `json:"opened_issues"`
	ClosedIssues        int                   `json:"closed_issues"`
	IssuesPerType       map[string]StatsCount `json:"issues_per_type"`
	IssuesPerStatus     map[string]StatsCount `json:"issues_per_status"`
	IssuesPerPriority   map[string]StatsCount `json:"issues_per_priority"`
	IssuesPerSeverity   map[string]StatsCount `json:"issues_per_severity"`
	IssuesPerOwner      map[string]StatsCount `json:"issues_per_owner"`
	IssuesPerAssignedTo map[string]StatsCount `json:"issues_per_assigned_to"`
	LastFourWeeks       LastFourWeeksStats    `json:"last_four_weeks_days"`
}

type ProjectMemberStats struct {
	ClosedBugs   map[string]int `json:"closed_bugs"`
	IocaineTasks map[string]int `json:"iocaine_tasks"`
	WikiChanges  map[string]int `json:"wiki_changes"`
	CreatedBugs  map[string]int `json:"created_bugs"`
	ClosedTasks  map[string]int `json:"closed_tasks"`
}

type SprintBurndownDay struct {
	Day           string  `json:"day"`
	Name          int     `json:"name"`
	OpenPoints    float64 `json:"open_points"`
	OptimalPoints float64 `json:"optimal_points"`
}

type SprintStats struct {
	Name                 string              `json:"name"`
	EstimatedStart       string              `json:"estimated_start"`
	EstimatedFinish      string              `json:"estimated_finish"`
	TotalPoints          map[string]any      `json:"total_points"`
	CompletedPoints      []any               `json:"completed_points"`
	TotalUserStories     int                 `json:"total_userstories"`
	CompletedUserStories int                 `json:"completed_userstories"`
	TotalTasks           int                 `json:"total_tasks"`
	CompletedTasks       int                 `json:"completed_tasks"`
	IocaineDoses         int                 `json:"iocaine_doses"`
	Days                 []SprintBurndownDay `json:"days"`
}

type GrowthStats struct {
	Total                      int            `json:"total"`
	Today                      int            `json:"today"`
	AverageLastSevenDays       float64        `json:"average_last_seven_days"`
	AverageLastFiveWorkingDays float64        `json:"average_last_five_working_days"`
	CountsLastYearPerWeek      map[string]int `json:"counts_last_year_per_week,omitempty"`
}

type PublicProjectStats struct {
	GrowthStats
	TotalWithBacklog          int     `json:"total_with_backlog"`
	PercentWithBacklog        float64 `json:"percent_with_backlog"`
	TotalWithKanban           int     `json:"total_with_kanban"`
	PercentWithKanban         float64 `json:"percent_with_kanban"`
	TotalWithBacklogAndKanban int     `json:"total_with_backlog_and_kanban"`
	PercentWithBoth           float64 `json:"percent_with_backlog_and_kanban"`
}

type SystemStats struct {
	Users       GrowthStats        `json:"users"`
	Projects    PublicProjectStats `json:"projects"`
	UserStories GrowthStats        `json:"userstories"`
}

type DiscoverStats struct {
	Projects struct {
		Total int `json:"total"`
	} `json:"projects"`
}

func (c *Client) GetProjectStats(ctx context.Context, projectID int64) (ProjectStats, error) {
	var result ProjectStats
	_, err := c.Get(ctx, fmt.Sprintf("projects/%d/stats", projectID), nil, &result)
	return result, err
}

func (c *Client) GetProjectIssueStats(ctx context.Context, projectID int64) (ProjectIssueStats, error) {
	var result ProjectIssueStats
	_, err := c.Get(ctx, fmt.Sprintf("projects/%d/issues_stats", projectID), nil, &result)
	return result, err
}

func (c *Client) GetProjectMemberStats(ctx context.Context, projectID int64) (ProjectMemberStats, error) {
	var result ProjectMemberStats
	_, err := c.Get(ctx, fmt.Sprintf("projects/%d/member_stats", projectID), nil, &result)
	return result, err
}

func (c *Client) GetSprintStats(ctx context.Context, sprintID int64) (SprintStats, error) {
	var result SprintStats
	_, err := c.Get(ctx, fmt.Sprintf("milestones/%d/stats", sprintID), nil, &result)
	return result, err
}

func (c *Client) GetSystemStats(ctx context.Context) (SystemStats, error) {
	var result SystemStats
	_, err := c.Get(ctx, "stats/system", nil, &result)
	return result, err
}

func (c *Client) GetDiscoverStats(ctx context.Context) (DiscoverStats, error) {
	var result DiscoverStats
	_, err := c.Get(ctx, "stats/discover", nil, &result)
	return result, err
}
