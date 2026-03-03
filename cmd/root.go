package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cwt",
	Short: "Crush Worktree Tool",
	Long:  "Creates git worktrees with full cmux dev environments — crush, lazygit, helix, and more.",
}

func Execute() error {
	return rootCmd.Execute()
}
