package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/Nishanth1812/devpulse/internal/models"
)

func RenderRepoTable(writer io.Writer, repos []models.RegisteredRepo) error {
	if len(repos) == 0 {
		_, err := fmt.Fprintln(writer, "No repositories registered")
		return err
	}

	table := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "REPO NAME\tABSOLUTE PATH"); err != nil {
		return err
	}
	for _, repo := range repos {
		if _, err := fmt.Fprintf(table, "%s\t%s\n", repo.Name, repo.Path); err != nil {
			return err
		}
	}

	return table.Flush()
}
