package cmd

import (
	"fmt"
	"os"

	"github.com/aschreifels/cwt/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up cwt with a guided configuration wizard",
	Long:  "Walks you through setting up cwt — branch prefix, project management provider, layout, and more.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cfg := config.DefaultConfig()

	existing, err := config.Load()
	if err == nil {
		cfg = existing
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a78bfa"))
	fmt.Println()
	fmt.Println(title.Render("  cwt init — cmux Worktree Tool Setup"))
	fmt.Println()

	var agent string
	var branchPrefix string
	var provider string
	var defaultProject string
	var editorCmd string
	var gitTool string

	agent = cfg.Defaults.Agent
	if agent == "" {
		agent = config.AgentCrush
	}
	branchPrefix = cfg.Defaults.BranchPrefix
	if cfg.ProjectManagement.Provider == "" || cfg.ProjectManagement.Provider == "none" {
		provider = "none"
	} else {
		provider = cfg.ProjectManagement.Provider
	}
	defaultProject = cfg.ProjectManagement.DefaultProject

	editorCmd = "hx ."
	gitTool = "lazygit"
	sidePanes := cfg.SidePanes()
	if len(sidePanes) > 0 {
		gitTool = sidePanes[0].Command
	}
	if len(sidePanes) > 1 {
		editorCmd = sidePanes[1].Command
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("AI agent").
				Description("Which AI coding agent do you use?").
				Options(
					huh.NewOption("Crush", config.AgentCrush),
					huh.NewOption("Claude Code", config.AgentClaude),
				).
				Value(&agent),

			huh.NewInput().
				Title("Branch prefix").
				Description("Prepended to all branch names (e.g. your initials). Leave empty for none.").
				Placeholder("initials").
				Value(&branchPrefix),

			huh.NewSelect[string]().
				Title("Project management provider").
				Description("Which tool do you use for tickets/issues?").
				Options(
					huh.NewOption("Linear", "linear"),
					huh.NewOption("GitHub Issues", "github"),
					huh.NewOption("Jira", "jira"),
					huh.NewOption("None", "none"),
				).
				Value(&provider),

			huh.NewInput().
				Title("Default project/team key").
				Description("Used when creating draft tickets (e.g. PROJ, BACKEND, INFRA).").
				Placeholder("PROJ").
				Value(&defaultProject),
		),

		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Git tool").
				Description("TUI for the top-right pane.").
				Options(
					huh.NewOption("lazygit", "lazygit"),
					huh.NewOption("gitui", "gitui"),
					huh.NewOption("tig", "tig"),
					huh.NewOption("Custom", "custom"),
				).
				Value(&gitTool),

			huh.NewInput().
				Title("Editor command").
				Description("Command for the bottom-right pane.").
				Placeholder("hx .").
				Value(&editorCmd),
		),
	)

	err = form.Run()
	if err != nil {
		return err
	}

	if gitTool == "custom" {
		customForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Custom git tool command").
					Placeholder("lazygit").
					Value(&gitTool),
			),
		)
		if err := customForm.Run(); err != nil {
			return err
		}
	}

	cfg.Defaults.Agent = agent
	cfg.Defaults.BranchPrefix = branchPrefix
	cfg.ProjectManagement.Provider = provider
	cfg.ProjectManagement.DefaultProject = defaultProject

	defaults := config.DefaultConfigForAgent(agent)
	if cfg.ProjectManagement.Prompts.Fetch == "" {
		cfg.ProjectManagement.Prompts.Fetch = defaults.ProjectManagement.Prompts.Fetch
	}
	if cfg.ProjectManagement.Prompts.Create == "" {
		cfg.ProjectManagement.Prompts.Create = defaults.ProjectManagement.Prompts.Create
	}

	agentDefaults := config.DefaultConfigForAgent(agent)
	mainPane := agentDefaults.Layout.Panes[0]
	cfg.Layout.Panes = []config.PaneConfig{
		mainPane,
		{Name: gitTool, Command: gitTool, Split: "right"},
		{Name: "editor", Command: editorCmd, Split: "down"},
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	success := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))

	fmt.Println()
	fmt.Println(success.Render("  Config saved ✓"))
	fmt.Println(dim.Render(fmt.Sprintf("  %s", config.ConfigPath())))
	fmt.Println()
	fmt.Println("  Get started:")
	fmt.Println(dim.Render("    cwt spawn my-feature"))
	fmt.Println(dim.Render("    cwt spawn my-feature -t PROJ-123"))
	fmt.Println(dim.Render("    cwt spawn my-feature --draft"))
	fmt.Println()

	return nil
}

func ConfigExists() bool {
	_, err := os.Stat(config.ConfigPath())
	return err == nil
}
