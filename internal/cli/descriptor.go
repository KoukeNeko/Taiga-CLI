package cli

import "sort"

const jsonSchemaDraft = "https://json-schema.org/draft/2020-12/schema"

type Safety string
type Idempotency string

const (
	SafetyRead    Safety      = "read"
	SafetyWrite   Safety      = "write"
	SafetyLocal   Safety      = "local"
	Idempotent    Idempotency = "idempotent"
	NonIdempotent Idempotency = "non_idempotent"
)

type Descriptor struct {
	Command     string         `json:"command"`
	Description string         `json:"description"`
	Safety      Safety         `json:"safety"`
	Idempotency Idempotency    `json:"idempotency"`
	Input       map[string]any `json:"input_schema"`
	Output      map[string]any `json:"output_schema"`
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"$schema": jsonSchemaDraft, "type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func descriptors() map[string]Descriptor {
	stringProperty := map[string]any{"type": "string"}
	integerProperty := map[string]any{"type": "integer"}
	booleanProperty := map[string]any{"type": "boolean"}
	commonOutput := objectSchema(map[string]any{"data": map[string]any{"type": "object"}, "meta": map[string]any{"type": "object"}}, "data", "meta")
	listOutput := objectSchema(map[string]any{"items": map[string]any{"type": "array"}, "page": map[string]any{"type": "object"}, "meta": map[string]any{"type": "object"}}, "items", "meta")
	return map[string]Descriptor{
		"version":           {Command: "version", Description: "Show CLI build information", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{}), Output: commonOutput},
		"doctor":            {Command: "doctor", Description: "Diagnose a Taiga endpoint and current context", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"host": stringProperty, "api_url": stringProperty}), Output: commonOutput},
		"auth login":        {Command: "auth login", Description: "Authenticate and save credentials", Safety: SafetyLocal, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"host": stringProperty, "api_url": stringProperty, "profile": stringProperty, "with_token": booleanProperty}), Output: commonOutput},
		"auth logout":       {Command: "auth logout", Description: "Remove saved credentials", Safety: SafetyLocal, Idempotency: Idempotent, Input: objectSchema(map[string]any{"profile": stringProperty}), Output: commonOutput},
		"auth status":       {Command: "auth status", Description: "Show current authentication status", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{}), Output: commonOutput},
		"config get":        {Command: "config get", Description: "Read a configuration value", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"key": stringProperty, "local": booleanProperty}, "key"), Output: commonOutput},
		"config set":        {Command: "config set", Description: "Write a configuration value", Safety: SafetyLocal, Idempotency: Idempotent, Input: objectSchema(map[string]any{"key": stringProperty, "value": stringProperty, "local": booleanProperty}, "key", "value"), Output: commonOutput},
		"config list":       {Command: "config list", Description: "List non-secret configuration", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"local": booleanProperty}), Output: commonOutput},
		"project list":      {Command: "project list", Description: "List projects", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"page": integerProperty, "limit": integerProperty}), Output: listOutput},
		"project view":      {Command: "project view", Description: "View a project", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"slug": stringProperty}, "slug"), Output: commonOutput},
		"project use":       {Command: "project use", Description: "Select a default project", Safety: SafetyLocal, Idempotency: Idempotent, Input: objectSchema(map[string]any{"slug": stringProperty, "local": booleanProperty}, "slug"), Output: commonOutput},
		"project create":    {Command: "project create", Description: "Create a project from a template", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"name": stringProperty, "description": stringProperty, "template": stringProperty, "public": booleanProperty, "dry_run": booleanProperty}, "name", "template"), Output: commonOutput},
		"project edit":      {Command: "project edit", Description: "Edit a project", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"slug": stringProperty, "name": stringProperty, "description": stringProperty, "public": booleanProperty, "private": booleanProperty, "epics": booleanProperty, "backlog": booleanProperty, "kanban": booleanProperty, "wiki": booleanProperty, "issues": booleanProperty, "dry_run": booleanProperty}, "slug"), Output: commonOutput},
		"project archive":   {Command: "project archive", Description: "Archive a project when supported by Taiga", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"slug": stringProperty}, "slug"), Output: commonOutput},
		"project unarchive": {Command: "project unarchive", Description: "Unarchive a project when supported by Taiga", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"slug": stringProperty}, "slug"), Output: commonOutput},
		"project delete":    {Command: "project delete", Description: "Request permanent project deletion", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"slug": stringProperty, "yes": booleanProperty, "dry_run": booleanProperty}, "slug"), Output: commonOutput},
		"epic list":         {Command: "epic list", Description: "List project epics", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"project": stringProperty, "page": integerProperty, "limit": integerProperty}), Output: listOutput},
		"epic view":         {Command: "epic view", Description: "View an epic", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty}, "ref"), Output: commonOutput},
		"epic create":       {Command: "epic create", Description: "Create an epic", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"project": stringProperty, "subject": stringProperty, "description": stringProperty, "status": stringProperty, "assignee": stringProperty, "color": stringProperty, "dry_run": booleanProperty}, "subject"), Output: commonOutput},
		"epic edit":         {Command: "epic edit", Description: "Edit an epic with OCC", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "subject": stringProperty, "description": stringProperty, "status": stringProperty, "assignee": stringProperty, "color": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"epic close":        {Command: "epic close", Description: "Move an epic to a closed status", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "status": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"epic stories":      {Command: "epic stories", Description: "List stories linked to an epic", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty}, "ref"), Output: listOutput},
		"epic link":         {Command: "epic link", Description: "Link a Story to an epic", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "story": stringProperty, "dry_run": booleanProperty}, "ref", "story"), Output: commonOutput},
		"epic unlink":       {Command: "epic unlink", Description: "Unlink a Story from an epic", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "story": stringProperty, "dry_run": booleanProperty}, "ref", "story"), Output: commonOutput},
		"epic watch":        {Command: "epic watch", Description: "Watch an epic", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"epic unwatch":      {Command: "epic unwatch", Description: "Stop watching an epic", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"epic history":      {Command: "epic history", Description: "Show epic activity and comments", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "type": stringProperty, "page": integerProperty, "limit": integerProperty}, "ref"), Output: listOutput},
		"issue list":        {Command: "issue list", Description: "List project issues", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"project": stringProperty, "page": integerProperty, "limit": integerProperty}), Output: listOutput},
		"issue view":        {Command: "issue view", Description: "View an issue", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty}, "ref"), Output: commonOutput},
		"issue create":      {Command: "issue create", Description: "Create an issue", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"project": stringProperty, "subject": stringProperty, "description": stringProperty, "status": stringProperty, "priority": stringProperty, "severity": stringProperty, "type": stringProperty, "assignee": stringProperty, "dry_run": booleanProperty}, "subject"), Output: commonOutput},
		"issue edit":        {Command: "issue edit", Description: "Edit an issue with OCC", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "subject": stringProperty, "description": stringProperty, "status": stringProperty, "priority": stringProperty, "severity": stringProperty, "type": stringProperty, "assignee": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"issue close":       {Command: "issue close", Description: "Move an issue to a closed status", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "status": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"issue assign":      {Command: "issue assign", Description: "Assign an issue", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "to": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref", "to"), Output: commonOutput},
		"issue comment":     {Command: "issue comment", Description: "Comment on an issue", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "body": stringProperty, "body_file": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"issue watch":       {Command: "issue watch", Description: "Watch an issue", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"issue unwatch":     {Command: "issue unwatch", Description: "Stop watching an issue", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"issue history":     {Command: "issue history", Description: "Show issue activity and comments", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "type": stringProperty, "page": integerProperty, "limit": integerProperty}, "ref"), Output: listOutput},
		"story list":        {Command: "story list", Description: "List project user stories", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"project": stringProperty, "sprint": stringProperty, "order_by": stringProperty, "page": integerProperty, "limit": integerProperty}), Output: listOutput},
		"story view":        {Command: "story view", Description: "View a user story", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty}, "ref"), Output: commonOutput},
		"story create":      {Command: "story create", Description: "Create a user story", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"project": stringProperty, "subject": stringProperty, "description": stringProperty, "status": stringProperty, "sprint": stringProperty, "dry_run": booleanProperty}, "subject"), Output: commonOutput},
		"story edit":        {Command: "story edit", Description: "Edit a user story with OCC", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "subject": stringProperty, "description": stringProperty, "status": stringProperty, "sprint": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"story close":       {Command: "story close", Description: "Move a user story to a closed status", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "status": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"story move":        {Command: "story move", Description: "Move a user story to a sprint or backlog", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "sprint": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref", "sprint"), Output: commonOutput},
		"story assign":      {Command: "story assign", Description: "Replace user story assignees", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "to": map[string]any{"type": "array", "items": stringProperty}, "base_version": integerProperty, "dry_run": booleanProperty}, "ref", "to"), Output: commonOutput},
		"story comment":     {Command: "story comment", Description: "Comment on a user story", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "body": stringProperty, "body_file": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"story watch":       {Command: "story watch", Description: "Watch a user story", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"story unwatch":     {Command: "story unwatch", Description: "Stop watching a user story", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"story history":     {Command: "story history", Description: "Show user story activity and comments", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "type": stringProperty, "page": integerProperty, "limit": integerProperty}, "ref"), Output: listOutput},
		"task list":         {Command: "task list", Description: "List project tasks", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"project": stringProperty, "story": stringProperty, "order_by": stringProperty, "page": integerProperty, "limit": integerProperty}), Output: listOutput},
		"task view":         {Command: "task view", Description: "View a task", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty}, "ref"), Output: commonOutput},
		"task create":       {Command: "task create", Description: "Create a task", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"project": stringProperty, "story": stringProperty, "subject": stringProperty, "description": stringProperty, "status": stringProperty, "assignee": stringProperty, "dry_run": booleanProperty}, "subject"), Output: commonOutput},
		"task edit":         {Command: "task edit", Description: "Edit a task with OCC", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "subject": stringProperty, "description": stringProperty, "status": stringProperty, "story": stringProperty, "assignee": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"task done":         {Command: "task done", Description: "Move a task to a closed status", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "status": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"task reopen":       {Command: "task reopen", Description: "Move a task to an open status", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "status": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"task assign":       {Command: "task assign", Description: "Assign a task", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "to": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref", "to"), Output: commonOutput},
		"task unassign":     {Command: "task unassign", Description: "Remove a task assignee", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"task move":         {Command: "task move", Description: "Move a task to a Story, Sprint, or backlog", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "story": stringProperty, "sprint": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"task comment":      {Command: "task comment", Description: "Comment on a task", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "body": stringProperty, "body_file": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"task watch":        {Command: "task watch", Description: "Watch a task", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"task unwatch":      {Command: "task unwatch", Description: "Stop watching a task", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"task history":      {Command: "task history", Description: "Show task activity and comments", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "type": stringProperty, "page": integerProperty, "limit": integerProperty}, "ref"), Output: listOutput},
		"sprint list":       {Command: "sprint list", Description: "List project sprints", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"project": stringProperty, "state": stringProperty, "order_by": stringProperty, "page": integerProperty, "limit": integerProperty}), Output: listOutput},
		"sprint view":       {Command: "sprint view", Description: "View a sprint", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"sprint": stringProperty}, "sprint"), Output: commonOutput},
		"sprint create":     {Command: "sprint create", Description: "Create a sprint", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"name": stringProperty, "start": stringProperty, "finish": stringProperty, "dry_run": booleanProperty}, "name", "start", "finish"), Output: commonOutput},
		"sprint edit":       {Command: "sprint edit", Description: "Edit a sprint", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"sprint": stringProperty, "name": stringProperty, "start": stringProperty, "finish": stringProperty, "dry_run": booleanProperty}, "sprint"), Output: commonOutput},
		"sprint close":      {Command: "sprint close", Description: "Close a sprint", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"sprint": stringProperty, "dry_run": booleanProperty}, "sprint"), Output: commonOutput},
		"sprint reopen":     {Command: "sprint reopen", Description: "Reopen a sprint", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"sprint": stringProperty, "dry_run": booleanProperty}, "sprint"), Output: commonOutput},
		"wiki list":         {Command: "wiki list", Description: "List project wiki pages", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"project": stringProperty, "page": integerProperty, "limit": integerProperty}), Output: listOutput},
		"wiki view":         {Command: "wiki view", Description: "View a wiki page", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty}, "ref"), Output: commonOutput},
		"wiki create":       {Command: "wiki create", Description: "Create a wiki page", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"project": stringProperty, "slug": stringProperty, "body": stringProperty, "body_file": stringProperty, "dry_run": booleanProperty}, "slug"), Output: commonOutput},
		"wiki edit":         {Command: "wiki edit", Description: "Edit a wiki page with OCC", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "slug": stringProperty, "body": stringProperty, "body_file": stringProperty, "base_version": integerProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"wiki delete":       {Command: "wiki delete", Description: "Delete a wiki page", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "yes": booleanProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"wiki watch":        {Command: "wiki watch", Description: "Watch a wiki page", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"wiki unwatch":      {Command: "wiki unwatch", Description: "Stop watching a wiki page", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "dry_run": booleanProperty}, "ref"), Output: commonOutput},
		"wiki history":      {Command: "wiki history", Description: "Show wiki page activity and comments", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"ref": stringProperty, "type": stringProperty, "page": integerProperty, "limit": integerProperty}, "ref"), Output: listOutput},
		"search":            {Command: "search", Description: "Search work items in a project", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"text": stringProperty, "type": stringProperty, "limit": integerProperty}, "text"), Output: listOutput},
		"attachment list":   {Command: "attachment list", Description: "List work item attachments", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"resource": stringProperty, "ref": stringProperty}, "resource", "ref"), Output: listOutput},
		"attachment view":   {Command: "attachment view", Description: "View attachment metadata", Safety: SafetyRead, Idempotency: Idempotent, Input: objectSchema(map[string]any{"resource": stringProperty, "id": integerProperty}, "resource", "id"), Output: commonOutput},
		"attachment add":    {Command: "attachment add", Description: "Upload a work item attachment", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"resource": stringProperty, "ref": stringProperty, "file": stringProperty, "name": stringProperty, "description": stringProperty, "deprecated": booleanProperty, "dry_run": booleanProperty}, "resource", "ref", "file"), Output: commonOutput},
		"attachment edit":   {Command: "attachment edit", Description: "Edit attachment metadata", Safety: SafetyWrite, Idempotency: NonIdempotent, Input: objectSchema(map[string]any{"resource": stringProperty, "id": integerProperty, "description": stringProperty, "deprecated": booleanProperty, "dry_run": booleanProperty}, "resource", "id"), Output: commonOutput},
		"attachment delete": {Command: "attachment delete", Description: "Delete an attachment", Safety: SafetyWrite, Idempotency: Idempotent, Input: objectSchema(map[string]any{"resource": stringProperty, "id": integerProperty, "yes": booleanProperty, "dry_run": booleanProperty}, "resource", "id"), Output: commonOutput},
	}
}

func descriptorNames(values map[string]Descriptor) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
