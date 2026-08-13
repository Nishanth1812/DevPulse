package cmd

import (
	"fmt"

	"github.com/Nishanth1812/devpulse/internal/services"
	"github.com/spf13/cobra"
)

var noteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage per-repository notes",
}

// resolveNoteRepo resolves a (possibly partial) repo name against the fuzzy
// matcher, printing candidates when ambiguous.
func resolveNoteRepo(cmd *cobra.Command, query string) (string, error) {
	result, err := fuzzyMatch(query)
	if err != nil {
		return "", err
	}
	if len(result.Candidates) > 0 {
		printCandidates(cmd.OutOrStdout(), query, result.Candidates)
		return "", nil
	}
	return result.Matched, nil
}

// noteRepoCompletion offers registered repo names for the repo argument.
func noteRepoCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	repos := manager.ListRepositories()
	var names []string
	for _, r := range repos {
		names = append(names, r.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

var noteAddCmd = &cobra.Command{
	Use:  "add <repo> <text>",
	Args: cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return noteRepoCompletion(cmd, args, toComplete)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := resolveNoteRepo(cmd, args[0])
		if err != nil {
			return err
		}
		if repo == "" {
			return nil
		}

		service := services.NoteService{}
		if err := service.Add(repo, args[1]); err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Note added\n")
		return err
	},
}

var noteListCmd = &cobra.Command{
	Use:  "list <repo>",
	Args: cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return noteRepoCompletion(cmd, args, toComplete)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := resolveNoteRepo(cmd, args[0])
		if err != nil {
			return err
		}
		if repo == "" {
			return nil
		}

		service := services.NoteService{}
		notes, err := service.List(repo)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), notes)
		return err
	},
}

func init() {
	noteCmd.AddCommand(noteAddCmd)
	noteCmd.AddCommand(noteListCmd)

	rootCmd.AddCommand(noteCmd)
}
