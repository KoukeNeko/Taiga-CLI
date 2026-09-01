package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) csvCommand() *cobra.Command {
	command := &cobra.Command{Use: "csv", Short: "Export project work items as CSV"}
	command.AddCommand(a.csvExportCommand())
	return command
}

func (a *App) csvExportCommand() *cobra.Command {
	var output string
	var force, dryRun bool
	command := &cobra.Command{Use: "export <epic|story|task|issue>", Short: "Export one project resource to a verified CSV file", Args: exactArgs(1), ValidArgs: []string{"epic", "story", "task", "issue"}, RunE: func(cmd *cobra.Command, args []string) error {
		resource := strings.ToLower(strings.TrimSpace(args[0]))
		if resource != "epic" && resource != "story" && resource != "task" && resource != "issue" {
			return usageError("CSV resource must be epic, story, task, or issue")
		}
		client, project, err := a.selectedProject(cmd.Context())
		if err != nil {
			return err
		}
		pathValue := output
		if pathValue == "" {
			pathValue = project.Slug + "-" + resource + ".csv"
		}
		path, err := filepath.Abs(pathValue)
		if err != nil {
			return err
		}
		if dryRun {
			return a.renderDryRun("export CSV", project.Slug+"#"+resource, map[string]any{"output": path, "temporary_server_token": true, "token_revoked_after_download": true})
		}
		if !force {
			if _, err := os.Stat(path); err == nil {
				return validationError("output_exists", "CSV output already exists; pass --force to replace it")
			}
		}
		uuid, err := client.CreateCSVToken(cmd.Context(), project.ID, resource)
		if err != nil {
			return err
		}
		if uuid == "" {
			return validationError("invalid_csv_token", "Taiga did not return a CSV token")
		}
		revoked := false
		defer func() {
			if !revoked {
				_ = client.RevokeCSVToken(context.WithoutCancel(cmd.Context()), project.ID, resource)
			}
		}()
		temporary, err := os.CreateTemp(filepath.Dir(path), ".taiga-csv-*.download")
		if err != nil {
			return err
		}
		temporaryPath := temporary.Name()
		defer func() { _ = os.Remove(temporaryPath) }()
		if err := temporary.Chmod(0o600); err != nil {
			_ = temporary.Close()
			return err
		}
		result, err := client.DownloadCSV(cmd.Context(), resource, uuid, temporary)
		if err != nil {
			_ = temporary.Close()
			return err
		}
		if err := temporary.Sync(); err != nil {
			_ = temporary.Close()
			return err
		}
		if err := temporary.Close(); err != nil {
			return err
		}
		if err := replaceLocalFile(temporaryPath, path, force); err != nil {
			return err
		}
		if err := client.RevokeCSVToken(cmd.Context(), project.ID, resource); err != nil {
			return fmt.Errorf("CSV downloaded but token revocation failed: %w", err)
		}
		revoked = true
		view := map[string]any{"resource": resource, "project": project.Slug, "path": path, "bytes": result.Bytes, "sha256": result.SHA256, "token_revoked": true, "verified": true}
		if a.global.JSON {
			return a.renderer().Data(view)
		}
		if !a.global.Quiet {
			_, _ = fmt.Fprintf(a.Out, "Exported %s CSV to %s (%d bytes, SHA-256 %s)\n", resource, path, result.Bytes, result.SHA256)
		}
		return nil
	}}
	command.Flags().StringVarP(&output, "output", "o", "", "output CSV file")
	command.Flags().BoolVar(&force, "force", false, "replace an existing output file")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "resolve and display the export without writing")
	return command
}
