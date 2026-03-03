package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aschreifels/cwt/skills"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	skillName   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a78bfa"))
	skillDesc   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	skillStatus = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	skillWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f59e0b"))
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage Crush skills bundled with cwt",
	Long: `Install or list the Crush skills that ship with cwt.

Skills are installed to ~/.config/crush/skills/ so Crush can
pick them up automatically (requires skills_paths in crush.json).`,
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available skills",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		all := skills.All()
		dir := skillsDir()

		fmt.Println()
		fmt.Println(skillName.Render("  Bundled Skills"))
		fmt.Println()

		for _, s := range all {
			installed := ""
			dest := filepath.Join(dir, s.Dir, "SKILL.md")
			if _, err := os.Stat(dest); err == nil {
				installed = skillStatus.Render("  (installed)")
			}

			fmt.Printf("  %-24s %s%s\n",
				skillName.Render(s.Name),
				skillDesc.Render(s.Description),
				installed,
			)
		}

		fmt.Println()
		fmt.Printf("  Install with: %s\n", skillDesc.Render("cwt skills install [name...]"))
		fmt.Println()
		return nil
	},
}

var skillsInstallForce bool

var skillsInstallCmd = &cobra.Command{
	Use:   "install [name...]",
	Short: "Install Crush skills to ~/.config/crush/skills/",
	Long: `Installs one or more bundled skills to the Crush skills directory.

With no arguments, installs all available skills.
Specify skill names to install selectively.

Existing skill files are skipped unless --force is used.`,
	Example: `  cwt skills install
  cwt skills install cmux-notifications
  cwt skills install cwt-orchestrator --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		all := skills.All()
		dir := skillsDir()

		toInstall := all
		if len(args) > 0 {
			toInstall = nil
			for _, name := range args {
				found := false
				for _, s := range all {
					if s.Name == name {
						toInstall = append(toInstall, s)
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("unknown skill: %s", name)
				}
			}
		}

		fmt.Println()
		for _, s := range toInstall {
			dest := filepath.Join(dir, s.Dir, "SKILL.md")

			if !skillsInstallForce {
				if _, err := os.Stat(dest); err == nil {
					fmt.Printf("  %s %s %s\n",
						skillWarn.Render("skip"),
						s.Name,
						skillDesc.Render("(already exists, use --force to overwrite)"),
					)
					continue
				}
			}

			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return fmt.Errorf("creating directory for %s: %w", s.Name, err)
			}

			if err := os.WriteFile(dest, s.Content, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", s.Name, err)
			}

			fmt.Printf("  %s %s → %s\n",
				skillStatus.Render("installed"),
				s.Name,
				skillDesc.Render(dest),
			)
		}
		fmt.Println()

		fmt.Printf("  %s\n", skillDesc.Render("Ensure crush.json includes skills_paths: [\"~/.config/crush/skills\"]"))
		fmt.Println()
		return nil
	},
}

func init() {
	skillsInstallCmd.Flags().BoolVarP(&skillsInstallForce, "force", "f", false, "Overwrite existing skill files")
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsInstallCmd)
	rootCmd.AddCommand(skillsCmd)
}

func skillsDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "crush", "skills")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "crush", "skills")
}
