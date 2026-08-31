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
