package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"text/tabwriter"

	"github.com/KoukeNeko/taiga-cli/internal/taiga"
	"github.com/spf13/cobra"
)

type attachmentTarget struct {
	Client    *taiga.Client
	ProjectID int64
	Resource  string
	ObjectID  int64
	Reference string
}

func (a *App) attachmentCommand() *cobra.Command {
	command := &cobra.Command{Use: "attachment", Aliases: []string{"attach"}, Short: "Work with Epic, Story, Task, Issue, and Wiki attachments"}
	command.AddCommand(a.attachmentListCommand(), a.attachmentViewCommand(), a.attachmentAddCommand(), a.attachmentEditCommand(), a.attachmentDeleteCommand())
	return command
}

func (a *App) attachmentListCommand() *cobra.Command {
	return &cobra.Command{
		Use: "list <epic|story|task|issue|wiki> <ref|slug>", Short: "List work item attachments", Args: exactArgs(2), ValidArgs: []string{"epic", "story", "task", "issue", "wiki"},
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := a.loadAttachmentTarget(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			attachments, err := target.Client.ListAttachments(cmd.Context(), target.Resource, target.ProjectID, target.ObjectID)
			if err != nil {
				return err
			}
			if a.global.JSON {
				return a.renderer().List(attachments, map[string]any{"total": len(attachments)})
			}
			writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(writer, "ID\tNAME\tSIZE\tDEPRECATED\tDESCRIPTION")
			for _, attachment := range attachments {
				_, _ = fmt.Fprintf(writer, "%d\t%s\t%d\t%t\t%s\n", attachment.ID, attachment.Name, attachment.Size, attachment.IsDeprecated, attachment.Description)
			}
			return writer.Flush()
		},
	}
}

func (a *App) attachmentViewCommand() *cobra.Command {
	return &cobra.Command{
		Use: "view <epic|story|task|issue|wiki> <id>", Short: "View attachment metadata", Args: exactArgs(2), ValidArgs: []string{"epic", "story", "task", "issue", "wiki"},
		RunE: func(cmd *cobra.Command, args []string) error {
			resource, id, err := parseAttachmentIdentity(args)
			if err != nil {
				return err
			}
			client, _, err := a.client(cmd.Context(), true)
			if err != nil {
				return err
			}
			attachment, err := client.GetAttachment(cmd.Context(), resource, id)
			if err != nil {
				return err
			}
			if a.global.JSON {
				return a.renderer().Data(attachment)
			}
			_, _ = fmt.Fprintf(a.Out, "%s\nID:          %d\nSize:        %d\nDeprecated:  %t\nDescription: %s\nURL:         %s\nSHA1:        %s\n", attachment.Name, attachment.ID, attachment.Size, attachment.IsDeprecated, attachment.Description, attachment.URL, attachment.SHA1)
			return nil
		},
	}
}

func (a *App) attachmentAddCommand() *cobra.Command {
	var name, description string
	var deprecated, dryRun bool
	command := &cobra.Command{
		Use: "add <epic|story|task|issue|wiki> <ref|slug> <file|->", Short: "Upload a work item attachment", Args: exactArgs(3), ValidArgs: []string{"epic", "story", "task", "issue", "wiki"},
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := a.loadAttachmentTarget(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			fileName := name
			var source io.Reader
			var file *os.File
			if args[2] == "-" {
				if fileName == "" {
					return usageError("--name is required when reading an attachment from stdin")
				}
				source = a.In
			} else {
				file, err = os.Open(args[2])
				if err != nil {
					return fmt.Errorf("open attachment: %w", err)
				}
				defer func() { _ = file.Close() }()
				source = file
				if fileName == "" {
					fileName = filepath.Base(args[2])
				}
			}
			if dryRun {
				return a.renderDryRun("upload attachment", target.Reference, map[string]any{"name": fileName, "description": description, "deprecated": deprecated})
			}
			attachment, err := target.Client.CreateAttachment(cmd.Context(), target.Resource, target.ProjectID, target.ObjectID, fileName, description, deprecated, source)
			if err != nil {
				return err
			}
			return a.renderAttachmentMutation("Uploaded", attachment)
		},
	}
	command.Flags().StringVar(&name, "name", "", "attachment filename; required for stdin")
	command.Flags().StringVar(&description, "description", "", "attachment description")
	command.Flags().BoolVar(&deprecated, "deprecated", false, "mark the attachment as deprecated")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "resolve and display the upload without writing")
	return command
}

func (a *App) attachmentEditCommand() *cobra.Command {
	var description string
	var deprecated, dryRun bool
	command := &cobra.Command{
		Use: "edit <epic|story|task|issue|wiki> <id>", Short: "Edit attachment metadata", Args: exactArgs(2), ValidArgs: []string{"epic", "story", "task", "issue", "wiki"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("description") && !cmd.Flags().Changed("deprecated") {
				return usageError("--description or --deprecated is required")
			}
			resource, id, err := parseAttachmentIdentity(args)
			if err != nil {
				return err
			}
			client, _, err := a.client(cmd.Context(), true)
			if err != nil {
				return err
			}
			request := taiga.UpdateAttachmentRequest{}
			if cmd.Flags().Changed("description") {
				request.Description = &description
			}
			if cmd.Flags().Changed("deprecated") {
				request.IsDeprecated = &deprecated
			}
			if dryRun {
				return a.renderDryRun("edit attachment", strconv.FormatInt(id, 10), map[string]any{"resource": resource, "description": request.Description, "deprecated": request.IsDeprecated})
			}
			attachment, err := client.UpdateAttachment(cmd.Context(), resource, id, request)
			if err != nil {
				return err
			}
			return a.renderAttachmentMutation("Updated", attachment)
		},
	}
	command.Flags().StringVar(&description, "description", "", "new attachment description")
	command.Flags().BoolVar(&deprecated, "deprecated", false, "set whether the attachment is deprecated")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "resolve and display the mutation without writing")
	return command
}

func (a *App) attachmentDeleteCommand() *cobra.Command {
	var yes, dryRun bool
	command := &cobra.Command{
		Use: "delete <epic|story|task|issue|wiki> <id>", Short: "Delete an attachment", Args: exactArgs(2), ValidArgs: []string{"epic", "story", "task", "issue", "wiki"},
		RunE: func(cmd *cobra.Command, args []string) error {
			resource, id, err := parseAttachmentIdentity(args)
			if err != nil {
				return err
			}
			client, _, err := a.client(cmd.Context(), true)
			if err != nil {
				return err
			}
			attachment, err := client.GetAttachment(cmd.Context(), resource, id)
			if err != nil {
				return err
			}
			if dryRun {
				return a.renderDryRun("delete attachment", strconv.FormatInt(id, 10), map[string]any{"resource": resource, "name": attachment.Name})
			}
			if !yes {
				if a.global.NoInput || !a.stdinTTY() {
					return confirmationRequired("attachment deletion requires --yes in non-interactive mode")
				}
				answer, err := a.readLine(fmt.Sprintf("Type %d to delete %s: ", id, attachment.Name))
				if err != nil {
					return err
				}
				if answer != strconv.FormatInt(id, 10) {
					return confirmationRequired("attachment deletion was not confirmed")
				}
			}
			if err := client.DeleteAttachment(cmd.Context(), resource, id); err != nil {
				return err
			}
			result := map[string]any{"id": id, "resource": resource, "deleted": true, "name": attachment.Name}
			if a.global.JSON {
				return a.renderer().Data(result)
			}
			if !a.global.Quiet {
				_, _ = fmt.Fprintf(a.Out, "Deleted attachment %d: %s\n", id, attachment.Name)
			}
			return nil
		},
	}
	command.Flags().BoolVar(&yes, "yes", false, "confirm attachment deletion")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "resolve and display the deletion without writing")
	return command
}

func (a *App) loadAttachmentTarget(ctx context.Context, resourceValue, refValue string) (attachmentTarget, error) {
	resource, err := taiga.NormalizeAttachmentResource(resourceValue)
	if err != nil {
		return attachmentTarget{}, usageError(err.Error())
	}
	target, err := a.loadActivityTarget(ctx, resource, refValue)
	if err != nil {
		return attachmentTarget{}, err
	}
	return attachmentTarget{Client: target.Client, ProjectID: target.ProjectID, Resource: resource, ObjectID: target.ID, Reference: target.reference()}, nil
}

func parseAttachmentIdentity(args []string) (string, int64, error) {
	resource, err := taiga.NormalizeAttachmentResource(args[0])
	if err != nil {
		return "", 0, usageError(err.Error())
	}
	id, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil || id <= 0 {
		return "", 0, usageError("attachment id must be a positive integer")
	}
	return resource, id, nil
}

func (a *App) renderAttachmentMutation(verb string, attachment taiga.Attachment) error {
	if a.global.JSON {
		return a.renderer().Data(attachment)
	}
	if !a.global.Quiet {
		_, _ = fmt.Fprintf(a.Out, "%s attachment %d: %s (%d bytes)\n", verb, attachment.ID, attachment.Name, attachment.Size)
	}
	return nil
}
