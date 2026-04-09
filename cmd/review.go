package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aschreifels/cwt/internal/config"
	"github.com/aschreifels/cwt/internal/git"
	"github.com/aschreifels/cwt/internal/tui"
	"github.com/aschreifels/cwt/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	reviewURL        string
	reviewNoCheckout bool
	reviewEditor     bool
)

var reviewCmd = &cobra.Command{
	Use:   "review <pr-number>",
	Short: "Open a review workspace for a GitHub pull request",
	Long: `Creates a cmux workspace focused on reviewing a pull request.

Fetches the PR metadata and diff, checks out the branch in a lightweight
worktree, and seeds your AI agent with the full PR context and review skills.

The PR number is required unless --url is provided with a full PR URL.
Uses the current repo by default, or specify a different repo via --url.`,
	Example: `  cwt review 42
  cwt review 42 --url https://github.com/org/repo
  cwt review --url https://github.com/org/repo/pull/42
  cwt review 42 --no-checkout`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var prNumber int
		var repo string

		if reviewURL != "" {
			parsedRepo, parsedPR, err := git.ParsePRFromURL(reviewURL)
			if err == nil && parsedPR > 0 {
				repo = parsedRepo
				if len(args) == 0 {
					prNumber = parsedPR
				}
			} else if len(args) == 0 {
				return fmt.Errorf("could not parse PR number from URL; provide it as an argument: cwt review <number> --url <repo-url>")
			} else {
				repo = extractRepoFromURL(reviewURL)
			}
		}

		if prNumber == 0 && len(args) > 0 {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}
			prNumber = n
		}

		if prNumber == 0 {
			return fmt.Errorf("PR number is required: cwt review <number>")
		}

		if !ConfigExists() {
			fmt.Println()
			fmt.Println("  No config found. Let's set one up first!")
			fmt.Println()
			if err := runInit(cmd, nil); err != nil {
				return err
			}
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		opts := workspace.ReviewOpts{
			PRNumber:     prNumber,
			Repo:         repo,
			NoCheckout:   reviewNoCheckout,
			SkipPrograms: !reviewEditor,
		}

		wsName := workspace.ReviewWorkspaceName(prNumber)
		branchDisplay := fmt.Sprintf("PR #%d", prNumber)

		var worktreeDir string
		if !reviewNoCheckout {
			dir, err := workspace.ResolveWorktreeDir(cfg, wsName)
			if err != nil {
				return err
			}
			worktreeDir = dir
		}

		updates := make(chan workspace.StepUpdate, 20)
		resultCh := make(chan tui.SpawnDone, 1)

		go func() {
			result, err := workspace.Review(cfg, opts, updates)
			var spawnResult *workspace.SpawnResult
			if result != nil {
				spawnResult = &workspace.SpawnResult{
					WorktreeDir: result.WorktreeDir,
					BranchName:  result.BranchName,
					WorkspaceID: result.WorkspaceID,
					Surfaces:    result.Surfaces,
				}
			}
			resultCh <- tui.SpawnDone{Result: spawnResult, Err: err}
		}()

		return tui.RunSpawn(wsName, branchDisplay, worktreeDir, "", updates, resultCh)
	},
}

func init() {
	rootCmd.AddCommand(reviewCmd)
	reviewCmd.Flags().StringVarP(&reviewURL, "url", "u", "", "GitHub repo URL or full PR URL")
	reviewCmd.Flags().BoolVar(&reviewNoCheckout, "no-checkout", false, "Skip worktree creation, review in current directory")
	reviewCmd.Flags().BoolVar(&reviewEditor, "editor", false, "Launch side panes (lazygit, editor) alongside the agent")
}

func extractRepoFromURL(url string) string {
	url = strings.TrimRight(url, "/")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "github.com/")

	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return url
}
