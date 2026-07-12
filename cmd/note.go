package cmd

import (
	"fmt"

	"github.com/Nishanth1812/devpulse/internal/services"
	"github.com/spf13/cobra"
)

var noteCmd = &cobra.Command{
	Use: "note",
}

var noteAddCmd = &cobra.Command{
	Use: "add <repo> <text>",
	RunE: func(cmd *cobra.Command, args []string) error {

		service := services.NoteService{}

		err := service.Add(args[0], args[1])
		if err != nil {
			return err
		}

		fmt.Println("Note added")

		return nil
	},
}

var noteListCmd = &cobra.Command{
	Use: "list <repo>",
	RunE: func(cmd *cobra.Command, args []string) error {

		service := services.NoteService{}

		notes, err := service.List(args[0])
		if err != nil {
			return err
		}

		fmt.Println(notes)

		return nil
	},
}

func init() {
	noteCmd.AddCommand(noteAddCmd)
	noteCmd.AddCommand(noteListCmd)

	rootCmd.AddCommand(noteCmd)
}
