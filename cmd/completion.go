package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/begray/agh/internal/project"
)

// fzfBashIntegration is appended to the standard bash completion output.
// It replaces the cobra `complete` binding with one that intercepts feature-name
// positions and opens an fzf picker; everything else falls through to cobra.
const fzfBashIntegration = `
# fzf integration for agh — requires fzf; degrades gracefully without it.
if command -v fzf >/dev/null 2>&1; then

__agh_fzf_list_features() {
    agh list 2>/dev/null | awk 'NR>1 {print $1}'
}

__agh_fzf_complete() {
    local subcommand="${COMP_WORDS[1]}"
    local cword="${COMP_CWORD}"

    # For commands whose first positional arg is a feature name, use fzf.
    case "$subcommand" in
        cd|stop|status|diff|exec)
            # cword==2 means we're completing the first positional arg.
            # exec also accepts args after --, but feature name is always pos 1.
            if [[ "$cword" -eq 2 ]]; then
                local features query selected
                features=$(__agh_fzf_list_features)
                [[ -z "$features" ]] && { __start_agh; return; }
                query="${COMP_WORDS[$cword]}"
                local fzf_opts=(--reverse --ansi
                    --preview 'agh status {} 2>/dev/null'
                    --preview-window 'right:50%:wrap'
                    --query "$query")
                # Use tmux popup when available for a compact picker
                if [[ -n "$TMUX" ]]; then
                    fzf_opts+=(--tmux center,60%,40%)
                fi
                selected=$(echo "$features" | fzf "${fzf_opts[@]}")
                if [[ -n "$selected" ]]; then
                    COMPREPLY=("$selected")
                    # Disable default file completion for this invocation
                    [[ $(type -t compopt) = "builtin" ]] && compopt +o default
                    # Send Device Status Report to force readline to redraw
                    # the prompt after fzf's TUI has overwritten it.
                    # This is the same trick fzf's own bash integration uses.
                    printf '\e[5n'
                    return
                fi
            fi
            ;;
    esac

    # Fall through to cobra-generated completion for everything else.
    __start_agh
}

# Replace the cobra binding with the fzf-aware wrapper.
if [[ $(type -t compopt) = "builtin" ]]; then
    complete -o default -F __agh_fzf_complete agh
else
    complete -o default -o nospace -F __agh_fzf_complete agh
fi

# Disable file completion for feature-name positions after fzf returns.
__agh_fzf_nospace() {
    if [[ $(type -t compopt) = "builtin" ]]; then
        compopt +o default
    fi
}

fi # end fzf guard
`

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate the autocompletion script for the specified shell",
	Long: `Generate the autocompletion script for agh for the specified shell.
See each sub-command's help for details on how to use the generated script.`,
}

var bashCompletionCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate the autocompletion script for bash",
	Long: `Generate the autocompletion script for the bash shell.

The script includes fzf integration when fzf is available: feature names
for cd, stop, status, diff, and exec are completed with an interactive fzf
picker (with agh status preview). Falls back to standard completion otherwise.

To load completions in your current shell session:

  source <(agh completion bash)

To load completions for every new session, add to your ~/.bashrc:

  eval "$(agh completion bash)"
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := rootCmd.GenBashCompletionV2(os.Stdout, true); err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, fzfBashIntegration)
		return nil
	},
}

var zshCompletionCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate the autocompletion script for zsh",
	Long: `Generate the autocompletion script for the zsh shell.

To load completions in your current shell session:

  source <(agh completion zsh)

To load completions for every new session, add to your ~/.zshrc:

  eval "$(agh completion zsh)"
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

var fishCompletionCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate the autocompletion script for fish",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenFishCompletion(os.Stdout, true)
	},
}

func init() {
	// Disable cobra's auto-generated completion command so we can own it fully.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	completionCmd.AddCommand(bashCompletionCmd, zshCompletionCmd, fishCompletionCmd)
	rootCmd.AddCommand(completionCmd)

	// Register feature name completion on relevant commands.
	stopCmd.ValidArgsFunction = completeFeatureNames
	statusCmd.ValidArgsFunction = completeFeatureNames
	diffCmd.ValidArgsFunction = completeFeatureNames
	execCmd.ValidArgsFunction = completeFeatureNames
}

// completeFeatureNames provides tab-completion for feature names (used by cobra
// for non-fzf shells and as fallback).
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
