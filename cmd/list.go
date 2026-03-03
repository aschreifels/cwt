package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/aschreifels/cwt/internal/git"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	listHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a78bfa"))
	listBranch  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	listPath    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	listBare    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f59e0b"))
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active git worktrees",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		worktrees, err := git.WorktreeList()
		if err != nil {
			return err
		}

		if len(worktrees) == 0 {
			fmt.Println("  No worktrees found")
			return nil
		}

		fmt.Println()
		fmt.Println(listHeader.Render("  Active Worktrees"))
		fmt.Println()

		for _, wt := range worktrees {
			if wt.Bare {
				fmt.Printf("  %s  %s\n",
					listBare.Render("(bare)"),
					listPath.Render(wt.Path),
				)
				continue
			}

			name := filepath.Base(wt.Path)
			branch := wt.Branch
			if branch == "" {
				branch = "(detached)"
			}

			fmt.Printf("  %-20s %s  %s\n",
				name,
				listBranch.Render(branch),
				listPath.Render(wt.Path),
			)
		}

		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
