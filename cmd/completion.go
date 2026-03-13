package cmd

import (
	"github.com/spf13/cobra"
	"github.com/begray/agh/internal/project"
)

// completeFeatureNames provides tab-completion for feature names.
func completeFeatureNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	proj, err := project.Detect()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	features, err := proj.ListFeatures()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, f := range features {
		names = append(names, f.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	// Register feature name completion on relevant commands
	stopCmd.ValidArgsFunction = completeFeatureNames
	statusCmd.ValidArgsFunction = completeFeatureNames
	diffCmd.ValidArgsFunction = completeFeatureNames
	execCmd.ValidArgsFunction = completeFeatureNames
}
