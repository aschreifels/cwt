package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aschreifels/cwt/internal/config"
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
	Short: "Manage agent skills bundled with cwt",
	Long: `Install or list the agent skills that ship with cwt.

For Crush: installs to ~/.config/crush/skills/ as SKILL.md files.
For Claude Code: installs to ~/.claude/CLAUDE.md as appended context.`,
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available skills",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		all := skills.All()
		dir := skillsDir()

		fmt.Println()
		fmt.Printf("  %s %s\n", skillName.Render("Bundled Skills"), skillDesc.Render(fmt.Sprintf("(agent: %s)", cfg.Defaults.Agent)))
		fmt.Println()

		for _, s := range all {
			installed := ""
			if cfg.IsClaude() {
				claudeMD := claudeMDPath()
				if data, err := os.ReadFile(claudeMD); err == nil && strings.Contains(string(data), s.Name) {
					installed = skillStatus.Render("  (installed)")
				}
			} else {
				dest := filepath.Join(dir, s.Dir, "SKILL.md")
				if _, err := os.Stat(dest); err == nil {
					installed = skillStatus.Render("  (installed)")
				}
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
	Short: "Install agent skills",
	Long: `Installs one or more bundled skills for the configured agent.

For Crush: installs as SKILL.md files to ~/.config/crush/skills/.
For Claude Code: appends skill content to ~/.claude/CLAUDE.md.

With no arguments, installs all available skills.
Specify skill names to install selectively.

Existing skill files are skipped unless --force is used.`,
	Example: `  cwt skills install
  cwt skills install cmux-notifications
  cwt skills install cwt-orchestrator --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		all := skills.All()

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
		if cfg.IsClaude() {
			return installSkillsClaude(toInstall)
		}
		return installSkillsCrush(toInstall)
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

func claudeMDPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "CLAUDE.md")
}

func installSkillsCrush(toInstall []skills.Skill) error {
	dir := skillsDir()
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
}

func installSkillsClaude(toInstall []skills.Skill) error {
	claudeMD := claudeMDPath()

	var existing []byte
	if data, err := os.ReadFile(claudeMD); err == nil {
		existing = data
	}

	var appended []string
	for _, s := range toInstall {
		marker := fmt.Sprintf("<!-- cwt-skill:%s -->", s.Name)
		if !skillsInstallForce && strings.Contains(string(existing), marker) {
			fmt.Printf("  %s %s %s\n",
				skillWarn.Render("skip"),
				s.Name,
				skillDesc.Render("(already in CLAUDE.md, use --force to overwrite)"),
			)
			continue
		}

		if skillsInstallForce && strings.Contains(string(existing), marker) {
			endMarker := fmt.Sprintf("<!-- /cwt-skill:%s -->", s.Name)
			start := strings.Index(string(existing), marker)
			end := strings.Index(string(existing), endMarker)
			if start >= 0 && end >= 0 {
				end += len(endMarker)
				if end < len(existing) && existing[end] == '\n' {
					end++
				}
				existing = append(existing[:start], existing[end:]...)
			}
		}

		endMarker := fmt.Sprintf("<!-- /cwt-skill:%s -->", s.Name)
		block := fmt.Sprintf("%s\n%s\n%s\n", marker, strings.TrimSpace(string(s.Content)), endMarker)
		appended = append(appended, block)

		fmt.Printf("  %s %s → %s\n",
			skillStatus.Render("installed"),
			s.Name,
			skillDesc.Render(claudeMD),
		)
	}

	if len(appended) > 0 {
		if err := os.MkdirAll(filepath.Dir(claudeMD), 0o755); err != nil {
			return fmt.Errorf("creating claude config dir: %w", err)
		}

		final := string(existing)
		if len(final) > 0 && !strings.HasSuffix(final, "\n") {
			final += "\n"
		}
		final += "\n" + strings.Join(appended, "\n")

		if err := os.WriteFile(claudeMD, []byte(final), 0o644); err != nil {
			return fmt.Errorf("writing CLAUDE.md: %w", err)
		}
	}

	fmt.Println()
	fmt.Printf("  %s\n", skillDesc.Render("Skills appended to "+claudeMD))
	fmt.Println()
	return nil
}
