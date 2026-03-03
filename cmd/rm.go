package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aschreifels/cwt/internal/config"
	"github.com/aschreifels/cwt/internal/git"
	"github.com/spf13/cobra"
)

var rmDeleteBranch bool

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a worktree and optionally delete the branch",
	Example: `  cwt rm my-feature
  cwt rm my-feature -D`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		var worktreeDir string
		if cfg.Defaults.WorktreeDir != "" {
			worktreeDir = filepath.Join(config.ExpandHome(cfg.Defaults.WorktreeDir), name)
		} else {
			base, err := git.DefaultWorktreeBase()
			if err != nil {
				return err
			}
			worktreeDir = filepath.Join(base, name)
		}

		var realBranch string
		if rmDeleteBranch {
			if info, err := os.Stat(worktreeDir); err == nil && info.IsDir() {
				realBranch, _ = git.BranchFromWorktree(worktreeDir)
			}
		}

		if info, err := os.Stat(worktreeDir); err == nil && info.IsDir() {
			fmt.Printf("  Removing worktree at %s...\n", worktreeDir)
			if err := git.WorktreeRemove(worktreeDir); err != nil {
				return err
			}
			fmt.Println("  ✓ Worktree removed")
		} else {
			fmt.Printf("  No worktree found at %s\n", worktreeDir)
		}

		if rmDeleteBranch && realBranch != "" {
			fmt.Printf("  Deleting branch '%s'...\n", realBranch)
			if err := git.DeleteBranch(realBranch); err != nil {
				return err
			}
			fmt.Println("  ✓ Branch deleted")
		}

		fmt.Println("\n  Done ✓")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&rmDeleteBranch, "delete-branch", "D", false, "Also delete the git branch")
}
