package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/KoukeNeko/taiga-cli/internal/config"
	"github.com/spf13/cobra"
)

func (a *App) projectCommand() *cobra.Command {
	command := &cobra.Command{Use: "project", Short: "Work with Taiga projects"}
	command.AddCommand(a.projectListCommand(), a.projectViewCommand(), a.projectUseCommand())
	return command
}

func (a *App) projectListCommand() *cobra.Command {
	var page, limit int
	command := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if limit < 1 || limit > 1000 || page < 1 {
				return usageError("--page must be positive and --limit must be between 1 and 1000")
			}
			client, _, err := a.client(cmd.Context(), true)
			if err != nil {
				return err
			}
			projects, pagination, err := client.ListProjects(cmd.Context(), page, limit)
			if err != nil {
				return err
			}
			if a.global.JSON {
				return a.renderer().List(projects, pagination)
			}
			writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(writer, "SLUG\tNAME\tPRIVATE\tARCHIVED")
			for _, project := range projects {
				_, _ = fmt.Fprintf(writer, "%s\t%s\t%t\t%t\n", project.Slug, project.Name, project.IsPrivate, project.IsArchived)
			}
			return writer.Flush()
		},
	}
	command.Flags().IntVar(&page, "page", 1, "page number")
	command.Flags().IntVar(&limit, "limit", 30, "maximum projects to return")
	return command
}

func (a *App) projectViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view <slug>",
		Short: "View a project",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := a.client(cmd.Context(), true)
			if err != nil {
				return err
			}
			project, err := client.GetProjectBySlug(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if a.global.JSON {
				return a.renderer().Data(project)
			}
			_, _ = fmt.Fprintf(a.Out, "%s\nSlug: %s\nPrivate: %t\nArchived: %t\n\n%s\n", project.Name, project.Slug, project.IsPrivate, project.IsArchived, project.Description)
			return nil
		},
	}
}

func (a *App) projectUseCommand() *cobra.Command {
	var local bool
	command := &cobra.Command{
		Use:   "use <slug>",
		Short: "Select a default project",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, settings, err := a.client(cmd.Context(), true)
			if err != nil {
				return err
			}
			project, err := client.GetProjectBySlug(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if local {
				if err := a.GitLocal.Set(cmd.Context(), "profile", settings.Profile); err != nil {
					if err == config.ErrNotGitRepository {
						return validationError("not_git_repository", err.Error())
					}
					return err
				}
				if err := a.GitLocal.Set(cmd.Context(), "project", project.Slug); err != nil {
					return err
				}
			} else {
				cfg, err := a.Config.Load()
				if err != nil {
					return err
				}
				profile := ensureProfile(&cfg, settings.Profile)
				profile.Project = project.Slug
				cfg.Profiles[settings.Profile] = profile
				if err := a.Config.Save(cfg); err != nil {
					return err
				}
			}
			result := map[string]any{"profile": settings.Profile, "project": project, "local": local}
			if a.global.JSON {
				return a.renderer().Data(result)
			}
			if !a.global.Quiet {
				_, _ = fmt.Fprintf(a.Out, "Using project %s (%s)%s\n", project.Name, project.Slug, map[bool]string{true: " for this Git repository", false: ""}[local])
			}
			return nil
		},
	}
	command.Flags().BoolVar(&local, "local", false, "save profile/project mapping in .git/config")
	return command
}
