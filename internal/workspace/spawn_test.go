package workspace

import (
	"strings"
	"testing"

	"github.com/aschreifels/cwt/internal/config"
)

func TestBuildBranchName(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config.Config
		opts   SpawnOpts
		want   string
	}{
		{
			name: "simple name no prefix",
			cfg:  config.Config{},
			opts: SpawnOpts{Name: "my-feature"},
			want: "my-feature",
		},
		{
			name: "with prefix",
			cfg:  config.Config{Defaults: config.DefaultsConfig{BranchPrefix: "jd"}},
			opts: SpawnOpts{Name: "my-feature"},
			want: "jd/my-feature",
		},
		{
			name: "with prefix and ticket",
			cfg:  config.Config{Defaults: config.DefaultsConfig{BranchPrefix: "jd"}},
			opts: SpawnOpts{Name: "my-feature", Ticket: "PROJ-123"},
			want: "jd/PROJ-123_my-feature",
		},
		{
			name: "ticket without prefix",
			cfg:  config.Config{},
			opts: SpawnOpts{Name: "my-feature", Ticket: "PROJ-123"},
			want: "PROJ-123_my-feature",
		},
		{
			name: "existing branch returns name",
			cfg:  config.Config{Defaults: config.DefaultsConfig{BranchPrefix: "jd"}},
			opts: SpawnOpts{Name: "my-feature", Existing: true},
			want: "my-feature",
		},
		{
			name: "existing with explicit branch",
			cfg:  config.Config{},
			opts: SpawnOpts{Name: "my-feature", Existing: true, ExistBranch: "fix/urgent"},
			want: "fix/urgent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildBranchName(tt.cfg, tt.opts)
			if got != tt.want {
				t.Errorf("BuildBranchName: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveBaseBranch(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		opts SpawnOpts
		want string
	}{
		{
			name: "explicit base from opts",
			cfg:  config.Config{},
			opts: SpawnOpts{BaseBranch: "develop"},
			want: "develop",
		},
		{
			name: "base from config",
			cfg:  config.Config{Defaults: config.DefaultsConfig{BaseBranch: "staging"}},
			opts: SpawnOpts{},
			want: "staging",
		},
		{
			name: "opts overrides config",
			cfg:  config.Config{Defaults: config.DefaultsConfig{BaseBranch: "staging"}},
			opts: SpawnOpts{BaseBranch: "develop"},
			want: "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveBaseBranch(tt.cfg, tt.opts)
			if got != tt.want {
				t.Errorf("ResolveBaseBranch: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveWorktreeDir(t *testing.T) {
	cfg := config.Config{
		Defaults: config.DefaultsConfig{
			WorktreeDir: "/tmp/worktrees",
		},
	}

	got, err := ResolveWorktreeDir(cfg, "my-feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/tmp/worktrees/my-feature"
	if got != want {
		t.Errorf("ResolveWorktreeDir: got %q, want %q", got, want)
	}
}

func TestResolvePrompt(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ProjectManagement.Provider = "linear"
	cfg.ProjectManagement.DefaultProject = "PROJ"

	t.Run("returns crush skill loading prompt when no project management", func(t *testing.T) {
		noCfg := config.DefaultConfig()
		got := resolvePrompt(noCfg, SpawnOpts{Ticket: "PROJ-1"})
		if got != crushSkillLoadingPrompt {
			t.Errorf("expected crush skill loading prompt only, got %q", got)
		}
	})

	t.Run("returns fetch prompt with skill loading suffix for ticket", func(t *testing.T) {
		got := resolvePrompt(cfg, SpawnOpts{Ticket: "PROJ-1", Name: "feat"})
		if got == "" {
			t.Error("expected non-empty fetch prompt")
		}
		if !strings.Contains(got, "PROJ-1") {
			t.Error("expected prompt to contain ticket ID")
		}
		if !strings.Contains(got, crushSkillLoadingPrompt) {
			t.Error("expected prompt to contain skill loading suffix")
		}
	})

	t.Run("returns create prompt with skill loading suffix for draft", func(t *testing.T) {
		got := resolvePrompt(cfg, SpawnOpts{CreateDraft: true, Name: "feat"})
		if got == "" {
			t.Error("expected non-empty create prompt")
		}
		if !strings.Contains(got, crushSkillLoadingPrompt) {
			t.Error("expected prompt to contain skill loading suffix")
		}
	})

	t.Run("returns skill loading prompt when no ticket or draft", func(t *testing.T) {
		got := resolvePrompt(cfg, SpawnOpts{Name: "feat"})
		if got != crushSkillLoadingPrompt {
			t.Errorf("expected crush skill loading prompt only, got %q", got)
		}
	})

	t.Run("always includes skill loading prompt", func(t *testing.T) {
		cases := []SpawnOpts{
			{Name: "feat"},
			{Name: "feat", Ticket: "PROJ-1"},
			{Name: "feat", CreateDraft: true},
		}
		for _, opts := range cases {
			got := resolvePrompt(cfg, opts)
			if !strings.Contains(got, crushSkillLoadingPrompt) {
				t.Errorf("opts %+v: expected crush skill loading prompt in result", opts)
			}
		}
	})

	t.Run("claude agent uses claude skill loading prompt", func(t *testing.T) {
		claudeCfg := config.DefaultConfigForAgent(config.AgentClaude)
		claudeCfg.ProjectManagement.Provider = "linear"
		claudeCfg.ProjectManagement.DefaultProject = "PROJ"

		got := resolvePrompt(claudeCfg, SpawnOpts{Ticket: "PROJ-1", Name: "feat"})
		if !strings.Contains(got, claudeSkillLoadingPrompt) {
			t.Error("expected claude skill loading prompt")
		}
		if strings.Contains(got, crushSkillLoadingPrompt) {
			t.Error("should not contain crush skill loading prompt")
		}
	})

	t.Run("claude agent returns claude prompt when no ticket", func(t *testing.T) {
		claudeCfg := config.DefaultConfigForAgent(config.AgentClaude)
		got := resolvePrompt(claudeCfg, SpawnOpts{Name: "feat"})
		if got != claudeSkillLoadingPrompt {
			t.Errorf("expected claude skill loading prompt, got %q", got)
		}
	})
}

func TestBuildMainCommand(t *testing.T) {
	t.Run("crush agent ignores prompt in command", func(t *testing.T) {
		cfg := config.DefaultConfig()
		pane := config.PaneConfig{Name: "crush", Command: "crush -c {{worktree_dir}}", Split: "main"}
		got := buildMainCommand(cfg, pane, "/tmp/wt", "do something")
		want := "crush -c /tmp/wt"
		if got != want {
			t.Errorf("buildMainCommand: got %q, want %q", got, want)
		}
	})

	t.Run("claude agent appends prompt to command", func(t *testing.T) {
		cfg := config.DefaultConfigForAgent(config.AgentClaude)
		pane := config.PaneConfig{Name: "claude", Command: "claude", Split: "main"}
		got := buildMainCommand(cfg, pane, "/tmp/wt", "do something")
		if got != "claude 'do something'" {
			t.Errorf("buildMainCommand: got %q", got)
		}
	})

	t.Run("claude agent with empty prompt", func(t *testing.T) {
		cfg := config.DefaultConfigForAgent(config.AgentClaude)
		pane := config.PaneConfig{Name: "claude", Command: "claude", Split: "main"}
		got := buildMainCommand(cfg, pane, "/tmp/wt", "")
		if got != "claude" {
			t.Errorf("buildMainCommand: got %q, want %q", got, "claude")
		}
	})
}

func TestExpandCommand(t *testing.T) {
	got := expandCommand("crush -c {{worktree_dir}}", "/tmp/worktrees/feat")
	want := "crush -c /tmp/worktrees/feat"
	if got != want {
		t.Errorf("expandCommand: got %q, want %q", got, want)
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"with space", "'with space'"},
		{"it's", "'it'\\''s'"},
	}

	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.want {
			t.Errorf("shellQuote(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}
