package cmd

import (
	"fmt"

	"github.com/aschreifels/cwt/internal/config"
	"github.com/aschreifels/cwt/internal/tui"
	"github.com/aschreifels/cwt/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	spawnBase     string
	spawnBranch   string
	spawnExisting bool
	spawnTicket   string
	spawnDraft    bool
	spawnNoEditor bool
)

var spawnCmd = &cobra.Command{
	Use:   "spawn <name>",
	Short: "Create a worktree with a full cmux dev workspace",
	Long: `Creates a git worktree and cmux workspace with configurable panes.

The --ticket flag fetches an existing ticket and seeds crush with its context.
The --draft flag creates a new draft ticket that gets updated as you work.
Both require project_management.provider to be set in config.

Default layout:
  ┌──────────────┬───────────┐
  │              │  lazygit  │
  │    crush     ├───────────┤
  │              │  helix .  │
  └──────────────┴───────────┘`,
	Example: `  cwt spawn my-feature
  cwt spawn my-feature -b develop
  cwt spawn my-feature -t PROJ-123
  cwt spawn my-feature --draft
  cwt spawn my-feature --existing
  cwt spawn hotfix --branch fix/urgent`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

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

		if (spawnTicket != "" || spawnDraft) && !cfg.HasProjectManagement() {
			return fmt.Errorf("project_management.provider must be set in %s to use --ticket or --draft", config.ConfigPath())
		}

		if spawnDraft && cfg.ProjectManagement.DefaultProject == "" {
			return fmt.Errorf("project_management.default_project must be set in %s to use --draft", config.ConfigPath())
		}

		opts := workspace.SpawnOpts{
			Name:         name,
			BaseBranch:   spawnBase,
			Existing:     spawnExisting || spawnBranch != "",
			ExistBranch:  spawnBranch,
			Ticket:       spawnTicket,
			CreateDraft:  spawnDraft,
			SkipPrograms: spawnNoEditor,
		}

		branchName := workspace.BuildBranchName(cfg, opts)
		worktreeDir, err := workspace.ResolveWorktreeDir(cfg, name)
		if err != nil {
			return err
		}

		ticketDisplay := spawnTicket
		if spawnDraft {
			ticketDisplay = "draft"
		}

		updates := make(chan workspace.StepUpdate, 20)
		resultCh := make(chan tui.SpawnDone, 1)

		go func() {
			result, err := workspace.Spawn(cfg, opts, updates)
			resultCh <- tui.SpawnDone{Result: result, Err: err}
		}()

		return tui.RunSpawn(name, branchName, worktreeDir, ticketDisplay, updates, resultCh)
	},
}

func init() {
	rootCmd.AddCommand(spawnCmd)
	spawnCmd.Flags().StringVarP(&spawnBase, "base", "b", "", "Base branch for new worktree (auto-detects main/master)")
	spawnCmd.Flags().StringVar(&spawnBranch, "branch", "", "Checkout an existing branch into the worktree")
	spawnCmd.Flags().BoolVar(&spawnExisting, "existing", false, "Checkout existing branch matching <name>")
	spawnCmd.Flags().StringVarP(&spawnTicket, "ticket", "t", "", "Ticket ID to fetch (e.g. PROJ-123)")
	spawnCmd.Flags().BoolVarP(&spawnDraft, "draft", "d", false, "Create a draft ticket and track work incrementally")
	spawnCmd.Flags().BoolVar(&spawnNoEditor, "no-editor", false, "Skip launching programs, just set up workspace")
}
