package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/KoukeNeko/taiga-cli/internal/taiga"
	"github.com/spf13/cobra"
)

const maxBatchItems = 1000

type batchCreatedView struct {
	Resource     string `json:"resource"`
	ID           int64  `json:"id"`
	Ref          int    `json:"ref"`
	Project      string `json:"project"`
	Subject      string `json:"subject"`
	Status       string `json:"status"`
	Version      int    `json:"version"`
	Sprint       string `json:"sprint,omitempty"`
	StoryRef     int    `json:"story_ref,omitempty"`
	StorySubject string `json:"story_subject,omitempty"`
}

type batchCreateOptions struct {
	Status string
	Sprint string
	Story  string
	Yes    bool
	DryRun bool
}

func (a *App) batchCommand() *cobra.Command {
	command := &cobra.Command{Use: "batch", Short: "Perform bounded bulk operations"}
	command.AddCommand(a.batchCreateCommand())
	return command
}

func (a *App) batchCreateCommand() *cobra.Command {
	options := batchCreateOptions{}
	command := &cobra.Command{
		Use: "create <epic|story|issue|task> <file|->", Short: "Create one work item per non-empty input line", Args: exactArgs(2), ValidArgs: []string{"epic", "story", "issue", "task"},
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := strings.ToLower(strings.TrimSpace(args[0]))
			if !validBatchResource(resource) {
				return usageError("resource must be epic, story, issue, or task")
			}
			if err := validateBatchOptions(resource, options); err != nil {
				return err
			}
			if args[1] == "-" && !options.DryRun && !options.Yes {
				return confirmationRequired("batch creation from stdin requires --yes so confirmation does not consume input data")
			}
			subjects, err := readBatchSubjects(a.In, args[1])
			if err != nil {
				return err
			}
			client, project, err := a.selectedProject(cmd.Context())
			if err != nil {
				return err
			}
			request, resolved, err := a.resolveBatchCreate(cmd.Context(), client, project, resource, subjects, options)
			if err != nil {
				return err
			}
			if options.DryRun {
				return a.renderDryRun("batch create "+resource, project.Slug, map[string]any{"count": len(subjects), "subjects": subjects, "status": resolved["status"], "sprint": resolved["sprint"], "story": resolved["story"], "maximum": maxBatchItems})
			}
			if !options.Yes {
				if a.global.NoInput || !a.stdinTTY() {
					return confirmationRequired("batch creation requires --yes in non-interactive mode")
				}
				answer, err := a.readLine(fmt.Sprintf("Type BATCH to create %d %s items: ", len(subjects), resource))
				if err != nil {
					return err
				}
				if answer != "BATCH" {
					return confirmationRequired("batch creation was not confirmed")
				}
			}
			created, err := client.BulkCreate(cmd.Context(), resource, request)
			if err != nil {
				return err
			}
			if len(created) != len(subjects) {
				return &taiga.Error{Kind: taiga.KindAmbiguousCommit, Operation: "bulk create " + resource, Message: fmt.Sprintf("Taiga returned %d of %d requested items; list the project before retrying", len(created), len(subjects)), Retryable: false, Details: map[string]any{"requested": len(subjects), "returned": len(created)}}
			}
			views := make([]batchCreatedView, 0, len(created))
			for _, item := range created {
				views = append(views, makeBatchCreatedView(resource, project.Slug, item))
			}
			page := map[string]any{"project": project.Slug, "resource": resource, "requested": len(subjects), "created": len(views), "verified": true}
			if a.global.JSON {
				return a.renderer().List(views, page)
			}
			writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(writer, "REF\tSUBJECT\tSTATUS\tSPRINT\tVERSION")
			for _, item := range views {
				_, _ = fmt.Fprintf(writer, "#%d\t%s\t%s\t%s\t%d\n", item.Ref, item.Subject, item.Status, item.Sprint, item.Version)
			}
			return writer.Flush()
		},
	}
	command.Flags().StringVar(&options.Status, "status", "", "common status name for every created item")
	command.Flags().StringVar(&options.Sprint, "sprint", "", "common Sprint for Issue or Task; Task requires one unless inferred from --story")
	command.Flags().StringVar(&options.Story, "story", "", "common parent Story for Task")
	command.Flags().BoolVar(&options.Yes, "yes", false, "confirm creation of every input item")
	command.Flags().BoolVar(&options.DryRun, "dry-run", false, "resolve metadata and display every planned item without writing")
	_ = command.RegisterFlagCompletionFunc("status", a.completeBatchStatuses)
	_ = command.RegisterFlagCompletionFunc("sprint", a.completeSprints)
	_ = command.RegisterFlagCompletionFunc("story", a.completeStories)
	return command
}

func validBatchResource(resource string) bool {
	switch resource {
	case "epic", "story", "issue", "task":
		return true
	default:
		return false
	}
}

func validateBatchOptions(resource string, options batchCreateOptions) error {
	if resource != "task" && options.Story != "" {
		return usageError("--story is only valid for task batches")
	}
	if resource != "issue" && resource != "task" && options.Sprint != "" {
		return usageError("--sprint is only valid for issue or task batches")
	}
	if strings.EqualFold(strings.TrimSpace(options.Sprint), "backlog") && resource == "task" {
		return usageError("Taiga bulk Task creation requires a real Sprint; backlog is not supported")
	}
	return nil
}

func readBatchSubjects(input io.Reader, path string) ([]string, error) {
	reader := input
	var file *os.File
	if path != "-" {
		opened, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open batch input: %w", err)
		}
		file, reader = opened, opened
		defer func() { _ = file.Close() }()
	}
	limited := &io.LimitedReader{R: reader, N: (4 << 20) + 1}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 64<<10), (4<<20)+1)
	subjects := make([]string, 0)
	for scanner.Scan() {
		subject := strings.TrimSpace(scanner.Text())
		if subject == "" {
			continue
		}
		subjects = append(subjects, subject)
		if len(subjects) > maxBatchItems {
			return nil, validationError("batch_too_large", fmt.Sprintf("batch contains more than %d items", maxBatchItems))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, validationError("invalid_batch_input", fmt.Sprintf("read batch input: %v", err))
	}
	if limited.N == 0 {
		return nil, validationError("batch_too_large", "batch input exceeds 4 MiB")
	}
	if len(subjects) == 0 {
		return nil, validationError("empty_batch", "batch input contains no non-empty subjects")
	}
	return subjects, nil
}

func (a *App) resolveBatchCreate(ctx context.Context, client *taiga.Client, project taiga.Project, resource string, subjects []string, options batchCreateOptions) (taiga.BulkCreateRequest, map[string]string, error) {
	request := taiga.BulkCreateRequest{ProjectID: project.ID, Subjects: strings.Join(subjects, "\n")}
	resolved := map[string]string{"status": "default", "sprint": "", "story": ""}
	if options.Status != "" {
		statusID, statusName, err := a.resolveBatchStatus(ctx, client, project.ID, resource, options.Status)
		if err != nil {
			return taiga.BulkCreateRequest{}, nil, err
		}
		request.StatusID, resolved["status"] = &statusID, statusName
	}
	if options.Sprint != "" && !strings.EqualFold(strings.TrimSpace(options.Sprint), "backlog") {
		sprint, err := a.resolveMilestone(ctx, client, project.ID, options.Sprint)
		if err != nil {
			return taiga.BulkCreateRequest{}, nil, err
		}
		request.MilestoneID, resolved["sprint"] = &sprint.ID, sprint.Slug
	}
	if resource == "issue" && strings.EqualFold(strings.TrimSpace(options.Sprint), "backlog") {
		resolved["sprint"] = "backlog"
	}
	if options.Story != "" {
		story, err := a.resolveParentStory(ctx, client, project.Slug, options.Story)
		if err != nil {
			return taiga.BulkCreateRequest{}, nil, err
		}
		if story.Milestone == nil {
			return taiga.BulkCreateRequest{}, nil, validationError("story_without_sprint", "Taiga bulk Task creation requires the parent Story to belong to a Sprint")
		}
		if request.MilestoneID != nil && *request.MilestoneID != *story.Milestone {
			return taiga.BulkCreateRequest{}, nil, validationError("sprint_story_mismatch", "--sprint must match the parent Story Sprint")
		}
		request.StoryID, request.MilestoneID = &story.ID, story.Milestone
		resolved["story"] = fmt.Sprintf("%s#%d", project.Slug, story.Ref)
		if resolved["sprint"] == "" {
			sprint, err := client.GetMilestone(ctx, *story.Milestone)
			if err != nil {
				return taiga.BulkCreateRequest{}, nil, err
			}
			resolved["sprint"] = sprint.Slug
		}
	}
	if resource == "task" && request.MilestoneID == nil {
		return taiga.BulkCreateRequest{}, nil, usageError("task batch creation requires --sprint or a parent --story that belongs to a Sprint")
	}
	return request, resolved, nil
}

func (a *App) resolveBatchStatus(ctx context.Context, client *taiga.Client, projectID int64, resource, value string) (int64, string, error) {
	switch resource {
	case "epic":
		status, err := a.resolveEpicStatus(ctx, client, projectID, value, false)
		return status.ID, status.Name, err
	case "story":
		status, err := a.resolveStoryStatus(ctx, client, projectID, value, false)
		return status.ID, status.Name, err
	case "issue":
		status, err := a.resolveIssueStatus(ctx, client, projectID, value, false)
		return status.ID, status.Name, err
	default:
		status, err := a.resolveTaskStatus(ctx, client, projectID, value, false)
		return status.ID, status.Name, err
	}
}

func makeBatchCreatedView(resource, project string, item taiga.BulkCreatedItem) batchCreatedView {
	view := batchCreatedView{Resource: resource, ID: item.ID, Ref: item.Ref, Project: project, Subject: item.Subject, Status: item.StatusExtraInfo.Name, Version: item.Version, Sprint: item.MilestoneSlug}
	if item.UserStoryExtraInfo != nil {
		view.StoryRef, view.StorySubject = item.UserStoryExtraInfo.Ref, item.UserStoryExtraInfo.Subject
	}
	return view
}

func (a *App) completeBatchStatuses(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	switch args[0] {
	case "epic":
		return a.completeEpicStatuses(cmd, nil, toComplete)
	case "story":
		return a.completeStoryStatuses(cmd, nil, toComplete)
	case "issue":
		return a.completeIssueStatuses(cmd, nil, toComplete)
	case "task":
		return a.completeTaskStatuses(cmd, nil, toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
