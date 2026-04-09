package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Defaults          DefaultsConfig          `toml:"defaults"`
	Layout            LayoutConfig            `toml:"layout"`
	ProjectManagement ProjectManagementConfig `toml:"project_management"`
	Review            ReviewConfig            `toml:"review"`
}

type DefaultsConfig struct {
	Agent        string `toml:"agent"`
	BranchPrefix string `toml:"branch_prefix"`
	BaseBranch   string `toml:"base_branch"`
	WorktreeDir  string `toml:"worktree_dir"`
}

type LayoutConfig struct {
	Panes []PaneConfig `toml:"panes"`
}

type PaneConfig struct {
	Name     string `toml:"name"`
	Command  string `toml:"command"`
	Split    string `toml:"split"`
	Disabled bool   `toml:"disabled"`
}

type ProjectManagementConfig struct {
	Provider       string        `toml:"provider"`
	DefaultProject string        `toml:"default_project"`
	Prompts        PromptConfig  `toml:"prompts"`
}

type PromptConfig struct {
	Fetch  string `toml:"fetch"`
	Create string `toml:"create"`
}

type ReviewConfig struct {
	Prompt string `toml:"prompt"`
}

const (
	AgentCrush  = "crush"
	AgentClaude = "claude"
)

func DefaultConfig() Config {
	return DefaultConfigForAgent(AgentCrush)
}

func DefaultConfigForAgent(agent string) Config {
	cfg := Config{
		Defaults: DefaultsConfig{
			Agent: agent,
		},
		ProjectManagement: ProjectManagementConfig{
			Provider: "none",
			Prompts: PromptConfig{
				Fetch: "Fetch the {{provider}} issue {{ticket}} using the {{provider}} MCP tools. " +
					"Read through the ticket title, description, comments, and any linked issues or documents. " +
					"Familiarize yourself with how this ticket relates to the codebase. " +
					"Then create a detailed plan of attack as a TODO list — outlining the files to change, " +
					"the approach, and any risks or open questions. " +
					"Wait for my review and confirmation before making any changes.",
				Create: "Create a draft {{provider}} issue in project {{project}} titled '{{name}}'. " +
					"Use the {{provider}} MCP tools to create it. " +
					"As you work on this feature, incrementally update the issue description with: " +
					"files changed, approach taken, and decisions made. " +
					"At the end of the session, finalize the issue with a proper title, description, " +
					"and acceptance criteria based on what was actually built.",
			},
		},
	}

	cfg.Review = ReviewConfig{
		Prompt: "Review this pull request thoroughly. Use your cwt-reviewer skill to route to the appropriate " +
			"language/domain-specific review skills based on the files changed. " +
			"Present the review in conversation first — do not post comments to GitHub unless I ask you to.",
	}

	switch agent {
	case AgentClaude:
		cfg.Layout = LayoutConfig{
			Panes: []PaneConfig{
				{Name: "claude", Command: "claude", Split: "main"},
				{Name: "lazygit", Command: "lazygit", Split: "right"},
				{Name: "helix", Command: "hx .", Split: "down"},
			},
		}
	default:
		cfg.Layout = LayoutConfig{
			Panes: []PaneConfig{
				{Name: "crush", Command: "crush -c {{worktree_dir}}", Split: "main"},
				{Name: "lazygit", Command: "lazygit", Split: "right"},
				{Name: "helix", Command: "hx .", Split: "down"},
			},
		}
	}

	return cfg
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "cwt")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cwt")
}

func ConfigPath() string {
	return filepath.Join(configDir(), "config.toml")
}

func Load() (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Defaults.Agent == "" {
		cfg.Defaults.Agent = AgentCrush
	}

	if len(cfg.Layout.Panes) == 0 {
		cfg.Layout.Panes = DefaultConfigForAgent(cfg.Defaults.Agent).Layout.Panes
	}

	defaults := DefaultConfig()
	if cfg.Review.Prompt == "" {
		cfg.Review.Prompt = defaults.Review.Prompt
	}
	if cfg.ProjectManagement.Prompts.Fetch == "" {
		cfg.ProjectManagement.Prompts.Fetch = defaults.ProjectManagement.Prompts.Fetch
	}
	if cfg.ProjectManagement.Prompts.Create == "" {
		cfg.ProjectManagement.Prompts.Create = defaults.ProjectManagement.Prompts.Create
	}

	return cfg, nil
}

func Save(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(ConfigPath(), data, 0o644)
}

func EnsureDefaults() error {
	if _, err := os.Stat(ConfigPath()); err == nil {
		return nil
	}
	return Save(DefaultConfig())
}

func (c Config) EnabledPanes() []PaneConfig {
	var panes []PaneConfig
	for _, p := range c.Layout.Panes {
		if !p.Disabled {
			panes = append(panes, p)
		}
	}
	return panes
}

func (c Config) MainPane() (PaneConfig, bool) {
	for _, p := range c.EnabledPanes() {
		if p.Split == "main" {
			return p, true
		}
	}
	return PaneConfig{}, false
}

func (c Config) SidePanes() []PaneConfig {
	var panes []PaneConfig
	for _, p := range c.EnabledPanes() {
		if p.Split != "main" {
			panes = append(panes, p)
		}
	}
	return panes
}

func (c Config) HasProjectManagement() bool {
	return c.ProjectManagement.Provider != "" && c.ProjectManagement.Provider != "none"
}

func (c Config) IsClaude() bool {
	return c.Defaults.Agent == AgentClaude
}

func (c Config) RenderPrompt(template, ticket, name string) string {
	r := strings.NewReplacer(
		"{{provider}}", c.ProjectManagement.Provider,
		"{{ticket}}", ticket,
		"{{project}}", c.ProjectManagement.DefaultProject,
		"{{name}}", name,
	)
	return r.Replace(template)
}

func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
