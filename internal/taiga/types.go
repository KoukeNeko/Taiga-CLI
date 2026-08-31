package taiga

type Page struct {
	Number int `json:"number"`
	Size   int `json:"size"`
	Total  int `json:"total"`
	Next   int `json:"next,omitempty"`
	Prev   int `json:"prev,omitempty"`
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name_display,omitempty"`
	Email    string `json:"email,omitempty"`
}

type AuthResponse struct {
	AuthToken    string `json:"auth_token"`
	RefreshToken string `json:"refresh"`
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	FullName     string `json:"full_name_display"`
}

type Project struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	IsPrivate   bool   `json:"is_private"`
	IsArchived  bool   `json:"is_archived,omitempty"`
}

type ExtraInfo struct {
	ID       int64  `json:"id"`
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
	FullName string `json:"full_name_display,omitempty"`
}

type Issue struct {
	ID                  int64      `json:"id"`
	Ref                 int        `json:"ref"`
	Project             int64      `json:"project"`
	Subject             string     `json:"subject"`
	Description         string     `json:"description,omitempty"`
	Version             int        `json:"version"`
	Status              int64      `json:"status"`
	StatusExtraInfo     ExtraInfo  `json:"status_extra_info"`
	Priority            int64      `json:"priority"`
	PriorityExtraInfo   ExtraInfo  `json:"priority_extra_info"`
	Severity            int64      `json:"severity"`
	SeverityExtraInfo   ExtraInfo  `json:"severity_extra_info"`
	Type                int64      `json:"type"`
	TypeExtraInfo       ExtraInfo  `json:"type_extra_info"`
	AssignedTo          *int64     `json:"assigned_to"`
	AssignedToExtraInfo *ExtraInfo `json:"assigned_to_extra_info,omitempty"`
	IsClosed            bool       `json:"is_closed"`
	CreatedDate         string     `json:"created_date,omitempty"`
	ModifiedDate        string     `json:"modified_date,omitempty"`
}

type IssueStatus struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IsClosed bool   `json:"is_closed"`
	Order    int    `json:"order"`
}

type NamedMetadata struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
}

type CreateIssueRequest struct {
	Project     int64    `json:"project"`
	Subject     string   `json:"subject"`
	Description string   `json:"description,omitempty"`
	Status      *int64   `json:"status,omitempty"`
	Priority    *int64   `json:"priority,omitempty"`
	Severity    *int64   `json:"severity,omitempty"`
	Type        *int64   `json:"type,omitempty"`
	AssignedTo  *int64   `json:"assigned_to,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type UpdateIssueRequest struct {
	Version     int     `json:"version"`
	Subject     *string `json:"subject,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *int64  `json:"status,omitempty"`
	Priority    *int64  `json:"priority,omitempty"`
	Severity    *int64  `json:"severity,omitempty"`
	Type        *int64  `json:"type,omitempty"`
	AssignedTo  *int64  `json:"assigned_to,omitempty"`
	Comment     *string `json:"comment,omitempty"`
}

type UserStory struct {
	ID              int64            `json:"id"`
	Ref             int              `json:"ref"`
	Project         int64            `json:"project"`
	Subject         string           `json:"subject"`
	Description     string           `json:"description,omitempty"`
	Version         int              `json:"version"`
	Status          int64            `json:"status"`
	StatusExtraInfo ExtraInfo        `json:"status_extra_info"`
	Milestone       *int64           `json:"milestone"`
	MilestoneSlug   string           `json:"milestone_slug,omitempty"`
	MilestoneName   string           `json:"milestone_name,omitempty"`
	AssignedUsers   []int64          `json:"assigned_users,omitempty"`
	TotalPoints     *float64         `json:"total_points,omitempty"`
	Points          map[string]int64 `json:"points,omitempty"`
	IsClosed        bool             `json:"is_closed"`
	IsBlocked       bool             `json:"is_blocked"`
	BlockedNote     string           `json:"blocked_note,omitempty"`
	BacklogOrder    int64            `json:"backlog_order"`
	SprintOrder     int64            `json:"sprint_order"`
	KanbanOrder     int64            `json:"kanban_order"`
	CreatedDate     string           `json:"created_date,omitempty"`
	ModifiedDate    string           `json:"modified_date,omitempty"`
}

type UserStoryStatus struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IsClosed bool   `json:"is_closed"`
	Order    int    `json:"order"`
}

type Milestone struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Project         int64  `json:"project"`
	Closed          bool   `json:"closed"`
	EstimatedStart  string `json:"estimated_start,omitempty"`
	EstimatedFinish string `json:"estimated_finish,omitempty"`
}

type CreateUserStoryRequest struct {
	Project       int64   `json:"project"`
	Subject       string  `json:"subject"`
	Description   string  `json:"description,omitempty"`
	Status        *int64  `json:"status,omitempty"`
	Milestone     *int64  `json:"milestone,omitempty"`
	AssignedUsers []int64 `json:"assigned_users,omitempty"`
}

type UpdateUserStoryRequest struct {
	Version       int      `json:"version"`
	Subject       *string  `json:"subject,omitempty"`
	Description   *string  `json:"description,omitempty"`
	Status        *int64   `json:"status,omitempty"`
	Milestone     **int64  `json:"milestone,omitempty"`
	AssignedUsers *[]int64 `json:"assigned_users,omitempty"`
	Comment       *string  `json:"comment,omitempty"`
}

type TaskStoryInfo struct {
	ID      int64  `json:"id"`
	Ref     int    `json:"ref"`
	Subject string `json:"subject"`
}

type Task struct {
	ID                  int64          `json:"id"`
	Ref                 int            `json:"ref"`
	Project             int64          `json:"project"`
	UserStory           *int64         `json:"user_story"`
	UserStoryExtraInfo  *TaskStoryInfo `json:"user_story_extra_info,omitempty"`
	Milestone           *int64         `json:"milestone"`
	MilestoneSlug       string         `json:"milestone_slug,omitempty"`
	Subject             string         `json:"subject"`
	Description         string         `json:"description,omitempty"`
	Version             int            `json:"version"`
	Status              int64          `json:"status"`
	StatusExtraInfo     ExtraInfo      `json:"status_extra_info"`
	AssignedTo          *int64         `json:"assigned_to"`
	AssignedToExtraInfo *ExtraInfo     `json:"assigned_to_extra_info,omitempty"`
	IsClosed            bool           `json:"is_closed"`
	IsBlocked           bool           `json:"is_blocked"`
	BlockedNote         string         `json:"blocked_note,omitempty"`
	CreatedDate         string         `json:"created_date,omitempty"`
	ModifiedDate        string         `json:"modified_date,omitempty"`
	FinishedDate        string         `json:"finished_date,omitempty"`
}

type TaskStatus struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IsClosed bool   `json:"is_closed"`
	Order    int    `json:"order"`
}

type CreateTaskRequest struct {
	Project     int64  `json:"project"`
	Subject     string `json:"subject"`
	Description string `json:"description,omitempty"`
	UserStory   *int64 `json:"user_story,omitempty"`
	Status      *int64 `json:"status,omitempty"`
	AssignedTo  *int64 `json:"assigned_to,omitempty"`
}

type UpdateTaskRequest struct {
	Version     int     `json:"version"`
	Subject     *string `json:"subject,omitempty"`
	Description *string `json:"description,omitempty"`
	UserStory   *int64  `json:"user_story,omitempty"`
	Status      *int64  `json:"status,omitempty"`
	AssignedTo  *int64  `json:"assigned_to,omitempty"`
	Comment     *string `json:"comment,omitempty"`
}
