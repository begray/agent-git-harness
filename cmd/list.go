package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/begray/agh/internal/project"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active features",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	proj, err := project.Detect()
	if err != nil {
		return fmt.Errorf("not in a git project: %w", err)
	}

	features, err := proj.ListFeatures()
	if err != nil {
		return err
	}

	if len(features) == 0 {
		fmt.Println("No active features")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "FEATURE\tBRANCH\tBASE\tPARENT\tIDE\tAI\tCREATED")
	for _, f := range features {
		parent := "-"
		if f.ParentFeature != "" {
			parent = f.ParentFeature
		}
		ide := "-"
		if f.IDE != "" {
			ide = f.IDE
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			f.Name, f.Branch, f.BaseBranch, parent, ide, f.AITool,
			f.CreatedAt.Format("2006-01-02 15:04"),
		)
	}
	w.Flush()
	return nil
}
