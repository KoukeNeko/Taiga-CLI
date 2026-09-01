package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type voteView struct {
	Resource string `json:"resource"`
	Project  string `json:"project"`
	Ref      int    `json:"ref"`
	Subject  string `json:"subject"`
	Voting   bool   `json:"voting"`
	Verified bool   `json:"verified"`
}

func (a *App) voteCommand(resource string, voting bool) *cobra.Command {
	name, short := "unvote", "Remove your vote from "+workItemArticle(resource)
	if voting {
		name, short = "vote", "Vote for "+workItemArticle(resource)
	}
	var dryRun bool
	command := &cobra.Command{Use: name + " <ref|project#ref|url>", Short: short, Args: exactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		target, err := a.loadActivityTarget(cmd.Context(), resource, args[0])
		if err != nil {
			return err
		}
		if dryRun {
			return a.renderDryRun(name+" "+resource, target.reference(), map[string]any{"voting": voting})
		}
		verified := target.IsVoter
		if verified != voting {
			verified, err = target.Client.SetVoting(cmd.Context(), resource, target.ID, voting)
			if err != nil {
				return err
			}
		}
		view := voteView{Resource: resource, Project: target.Project, Ref: target.Ref, Subject: target.Subject, Voting: verified, Verified: verified == voting}
		if a.global.JSON {
			return a.renderer().Data(view)
		}
		if !a.global.Quiet {
			_, _ = fmt.Fprintf(a.Out, "%s %s %s\n", map[bool]string{true: "Voted for", false: "Removed vote from"}[voting], resource, target.reference())
		}
		return nil
	}}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "resolve and display the vote change without writing")
	command.ValidArgsFunction = a.activityCompletion(resource)
	return command
}
