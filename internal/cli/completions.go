package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/KoukeNeko/taiga-cli/internal/taiga"
	"github.com/spf13/cobra"
)

const completionTimeout = 2 * time.Second

func (a *App) completionContext(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	return context.WithTimeout(cmd.Context(), completionTimeout)
}

func completionResult(values []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	needle := strings.ToLower(strings.TrimSpace(toComplete))
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		candidate := strings.SplitN(value, "\t", 2)[0]
		if needle == "" || strings.HasPrefix(strings.ToLower(candidate), needle) {
			filtered = append(filtered, value)
		}
	}
	sort.Strings(filtered)
	return filtered, cobra.ShellCompDirectiveNoFileComp
}

func noCompletions() ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func (a *App) completeProfiles(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := a.Config.Load()
	if err != nil {
		return noCompletions()
	}
	values := make([]string, 0, len(cfg.Profiles))
	for name, profile := range cfg.Profiles {
		values = append(values, fmt.Sprintf("%s\t%s", name, profile.APIURL))
	}
	return completionResult(values, toComplete)
}

func (a *App) completeProjects(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := a.completionContext(cmd)
	defer cancel()
	client, _, err := a.client(ctx, true)
	if err != nil {
		return noCompletions()
	}
	projects, _, err := client.ListProjects(ctx, 1, 100)
	if err != nil {
		return noCompletions()
	}
	values := make([]string, 0, len(projects))
	for _, project := range projects {
		values = append(values, fmt.Sprintf("%s\t%s", project.Slug, project.Name))
	}
	return completionResult(values, toComplete)
}

func (a *App) completionProject(ctx context.Context) (*taiga.Client, taiga.Project, error) {
	client, settings, err := a.client(ctx, true)
	if err != nil {
		return nil, taiga.Project{}, err
	}
	if settings.Project == "" {
		return nil, taiga.Project{}, validationError("missing_project", "no project selected")
	}
	project, err := client.GetProjectBySlug(ctx, settings.Project)
	return client, project, err
}

func (a *App) completeIssues(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := a.completionContext(cmd)
	defer cancel()
	client, project, err := a.completionProject(ctx)
	if err != nil {
		return noCompletions()
	}
	items, _, err := client.ListIssues(ctx, project.ID, 1, 100)
	if err != nil {
		return noCompletions()
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, fmt.Sprintf("%d\t%s", item.Ref, item.Subject))
	}
	return completionResult(values, toComplete)
}

func (a *App) completeStories(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := a.completionContext(cmd)
	defer cancel()
	client, project, err := a.completionProject(ctx)
	if err != nil {
		return noCompletions()
	}
	items, _, err := client.ListUserStories(ctx, project.ID, 1, 100, nil, false, "backlog_order")
	if err != nil {
		return noCompletions()
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, fmt.Sprintf("%d\t%s", item.Ref, item.Subject))
	}
	return completionResult(values, toComplete)
}

func (a *App) completeTasks(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := a.completionContext(cmd)
	defer cancel()
	client, project, err := a.completionProject(ctx)
	if err != nil {
		return noCompletions()
	}
	items, _, err := client.ListTasks(ctx, project.ID, nil, 1, 100, "us_order")
	if err != nil {
		return noCompletions()
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, fmt.Sprintf("%d\t%s", item.Ref, item.Subject))
	}
	return completionResult(values, toComplete)
}

func (a *App) completeWikiPages(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := a.completionContext(cmd)
	defer cancel()
	client, project, err := a.completionProject(ctx)
	if err != nil {
		return noCompletions()
	}
	pages, _, err := client.ListWikiPages(ctx, project.ID, 1, 100)
	if err != nil {
		return noCompletions()
	}
	values := make([]string, 0, len(pages))
	for _, page := range pages {
		values = append(values, fmt.Sprintf("%s\tversion %d", page.Slug, page.Version))
	}
	return completionResult(values, toComplete)
}

func (a *App) completeMembers(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := a.completionContext(cmd)
	defer cancel()
	client, project, err := a.completionProject(ctx)
	if err != nil {
		return noCompletions()
	}
	users, err := client.ListProjectUsers(ctx, project.ID)
	if err != nil {
		return noCompletions()
	}
	values := make([]string, 0, len(users))
	for _, user := range users {
		values = append(values, fmt.Sprintf("%s\t%s", user.Username, user.FullName))
	}
	return completionResult(values, toComplete)
}

func (a *App) completeSprints(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return a.completeSprintValues(cmd, toComplete, true)
}

func (a *App) completeSprintNames(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return a.completeSprintValues(cmd, toComplete, false)
}

func (a *App) completeSprintValues(cmd *cobra.Command, toComplete string, includeBacklog bool) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := a.completionContext(cmd)
	defer cancel()
	client, project, err := a.completionProject(ctx)
	if err != nil {
		return noCompletions()
	}
	milestones, err := client.Milestones(ctx, project.ID, nil)
	if err != nil {
		return noCompletions()
	}
	values := []string{}
	if includeBacklog {
		values = append(values, "backlog\tNo sprint")
	}
	for _, milestone := range milestones {
		values = append(values, fmt.Sprintf("%s\t%s", milestone.Slug, milestone.Name))
	}
	return completionResult(values, toComplete)
}

func (a *App) completeIssueStatuses(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := a.completionContext(cmd)
	defer cancel()
	client, project, err := a.completionProject(ctx)
	if err != nil {
		return noCompletions()
	}
	statuses, err := client.IssueStatuses(ctx, project.ID)
	if err != nil {
		return noCompletions()
	}
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, fmt.Sprintf("%s\tclosed=%t", status.Name, status.IsClosed))
	}
	return completionResult(values, toComplete)
}

func (a *App) completeStoryStatuses(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := a.completionContext(cmd)
	defer cancel()
	client, project, err := a.completionProject(ctx)
	if err != nil {
		return noCompletions()
	}
	statuses, err := client.UserStoryStatuses(ctx, project.ID)
	if err != nil {
		return noCompletions()
	}
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, fmt.Sprintf("%s\tclosed=%t", status.Name, status.IsClosed))
	}
	return completionResult(values, toComplete)
}

func (a *App) completeTaskStatuses(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := a.completionContext(cmd)
	defer cancel()
	client, project, err := a.completionProject(ctx)
	if err != nil {
		return noCompletions()
	}
	statuses, err := client.TaskStatuses(ctx, project.ID)
	if err != nil {
		return noCompletions()
	}
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, fmt.Sprintf("%s\tclosed=%t", status.Name, status.IsClosed))
	}
	return completionResult(values, toComplete)
}

func (a *App) completeNamedMetadata(loader func(context.Context, *taiga.Client, int64) ([]taiga.NamedMetadata, error)) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx, cancel := a.completionContext(cmd)
		defer cancel()
		client, project, err := a.completionProject(ctx)
		if err != nil {
			return noCompletions()
		}
		items, err := loader(ctx, client, project.ID)
		if err != nil {
			return noCompletions()
		}
		values := make([]string, 0, len(items))
		for _, item := range items {
			values = append(values, item.Name)
		}
		return completionResult(values, toComplete)
	}
}
