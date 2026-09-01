package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) projectLikeCommand(like bool) *cobra.Command {
	verb := "like"
	if !like {
		verb = "unlike"
	}
	var dryRun bool
	command := &cobra.Command{Use: verb + " [slug]", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, project, err := a.selectedProject(cmd.Context())
		if err != nil {
			return err
		}
		if len(args) == 1 && !strings.EqualFold(args[0], project.Slug) {
			project, err = client.GetProjectBySlug(cmd.Context(), args[0])
			if err != nil {
				return err
			}
		}
		if dryRun {
			return a.renderDryRun(verb+" project", project.Slug, nil)
		}
		if like {
			err = client.LikeProject(cmd.Context(), project.ID)
		} else {
			err = client.UnlikeProject(cmd.Context(), project.ID)
		}
		if err != nil {
			return err
		}
		label := "Liked"
		if !like {
			label = "Unliked"
		}
		return a.renderAdminMutation(label, "project", map[string]any{"project": project.Slug, "liked": like})
	}}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "display the mutation without writing")
	return command
}

func (a *App) projectFansCommand() *cobra.Command {
	var page, limit int
	command := &cobra.Command{Use: "fans [slug]", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, project, err := a.selectedProject(cmd.Context())
		if err != nil {
			return err
		}
		if len(args) == 1 && !strings.EqualFold(args[0], project.Slug) {
			project, err = client.GetProjectBySlug(cmd.Context(), args[0])
			if err != nil {
				return err
			}
		}
		values, paging, err := client.ProjectFans(cmd.Context(), project.ID, page, limit)
		if err != nil {
			return err
		}
		return a.renderer().List(values, map[string]any{"project": project.Slug, "page": paging})
	}}
	command.Flags().IntVar(&page, "page", 1, "page number")
	command.Flags().IntVar(&limit, "limit", 100, "items per page")
	return command
}

func (a *App) projectTransferCommand() *cobra.Command {
	command := &cobra.Command{Use: "transfer", Short: "Manage project ownership transfers"}
	command.AddCommand(a.projectTransferRequestCommand(), a.projectTransferStartCommand(), a.projectTransferTokenCommand("validate-token"), a.projectTransferTokenCommand("accept"), a.projectTransferTokenCommand("reject"))
	return command
}

func (a *App) projectTransferRequestCommand() *cobra.Command {
	var dryRun bool
	command := &cobra.Command{Use: "request", Args: exactArgs(0), RunE: func(cmd *cobra.Command, _ []string) error {
		client, project, err := a.selectedProject(cmd.Context())
		if err != nil {
			return err
		}
		if dryRun {
			return a.renderDryRun("request project transfer", project.Slug, nil)
		}
		if err := client.TransferProject(cmd.Context(), project.ID, "request", nil); err != nil {
			return err
		}
		return a.renderAdminMutation("Requested", "project transfer", map[string]any{"project": project.Slug, "requested": true})
	}}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "display the request without writing")
	return command
}

func (a *App) projectTransferStartCommand() *cobra.Command {
	var user int64
	var reason string
	var yes, dryRun bool
	command := &cobra.Command{Use: "start", Args: exactArgs(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if user <= 0 {
			return usageError("--user must be a positive member user ID")
		}
		client, project, err := a.selectedProject(cmd.Context())
		if err != nil {
			return err
		}
		if dryRun {
			return a.renderDryRun("start project transfer", project.Slug, map[string]any{"user": user, "reason": reason})
		}
		if !yes {
			return confirmationRequired("starting an ownership transfer requires --yes")
		}
		if err := client.TransferProject(cmd.Context(), project.ID, "start", map[string]any{"user": user, "reason": reason}); err != nil {
			return err
		}
		return a.renderAdminMutation("Started", "project transfer", map[string]any{"project": project.Slug, "target_user": user, "started": true})
	}}
	command.Flags().Int64Var(&user, "user", 0, "target member user ID")
	command.Flags().StringVar(&reason, "reason", "", "transfer reason")
	command.Flags().BoolVar(&yes, "yes", false, "confirm ownership transfer start")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "display the transfer without writing")
	return command
}

func (a *App) projectTransferTokenCommand(action string) *cobra.Command {
	var token, reason string
	var yes, dryRun bool
	command := &cobra.Command{Use: action, Args: exactArgs(0), RunE: func(cmd *cobra.Command, _ []string) error {
		if strings.TrimSpace(token) == "" {
			return usageError("--token is required")
		}
		client, project, err := a.selectedProject(cmd.Context())
		if err != nil {
			return err
		}
		body := map[string]any{"token": token}
		if reason != "" {
			body["reason"] = reason
		}
		if dryRun {
			return a.renderDryRun(action+" project transfer", project.Slug, map[string]any{"token": "[REDACTED]", "reason": reason})
		}
		if action != "validate-token" && !yes {
			return confirmationRequired(action + " project transfer requires --yes")
		}
		if err := client.TransferProject(cmd.Context(), project.ID, action, body); err != nil {
			return err
		}
		return a.renderAdminMutation("Completed", "project transfer "+action, map[string]any{"project": project.Slug, "action": action, "verified": true})
	}}
	command.Flags().StringVar(&token, "token", "", "transfer token (never shown in output)")
	command.Flags().StringVar(&reason, "reason", "", "acceptance or rejection reason")
	command.Flags().BoolVar(&yes, "yes", false, "confirm the transfer decision")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "display the transfer without writing")
	_ = command.Flags().MarkHidden("token")
	return command
}
