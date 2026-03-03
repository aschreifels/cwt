package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ProjectManagement.Provider != "none" {
		t.Errorf("expected default provider 'none', got %q", cfg.ProjectManagement.Provider)
	}

	if len(cfg.Layout.Panes) != 3 {
		t.Errorf("expected 3 default panes, got %d", len(cfg.Layout.Panes))
	}

	main, ok := cfg.MainPane()
	if !ok {
		t.Fatal("expected main pane to exist")
	}
	if main.Name != "crush" {
		t.Errorf("expected main pane name 'crush', got %q", main.Name)
	}

	if cfg.ProjectManagement.Prompts.Fetch == "" {
		t.Error("expected default fetch prompt to be non-empty")
	}
	if cfg.ProjectManagement.Prompts.Create == "" {
		t.Error("expected default create prompt to be non-empty")
	}
}

func TestHasProjectManagement(t *testing.T) {
	tests := []struct {
		provider string
		want     bool
	}{
		{"", false},
		{"none", false},
		{"linear", true},
		{"github", true},
		{"jira", true},
	}

	for _, tt := range tests {
		cfg := DefaultConfig()
		cfg.ProjectManagement.Provider = tt.provider
		if got := cfg.HasProjectManagement(); got != tt.want {
			t.Errorf("HasProjectManagement() with provider=%q: got %v, want %v", tt.provider, got, tt.want)
		}
	}
}

func TestEnabledPanes(t *testing.T) {
	cfg := DefaultConfig()
	enabled := cfg.EnabledPanes()
	if len(enabled) != 3 {
		t.Errorf("expected 3 enabled panes, got %d", len(enabled))
	}

	cfg.Layout.Panes[1].Disabled = true
	enabled = cfg.EnabledPanes()
	if len(enabled) != 2 {
		t.Errorf("expected 2 enabled panes after disabling one, got %d", len(enabled))
	}
}

func TestMainPane(t *testing.T) {
	cfg := DefaultConfig()
	main, ok := cfg.MainPane()
	if !ok {
		t.Fatal("expected main pane")
	}
	if main.Split != "main" {
		t.Errorf("expected split 'main', got %q", main.Split)
	}

	cfg.Layout.Panes = []PaneConfig{
		{Name: "side", Command: "echo", Split: "right"},
	}
	_, ok = cfg.MainPane()
	if ok {
		t.Error("expected no main pane when none configured")
	}
}

func TestSidePanes(t *testing.T) {
	cfg := DefaultConfig()
	sides := cfg.SidePanes()
	if len(sides) != 2 {
		t.Errorf("expected 2 side panes, got %d", len(sides))
	}
	for _, p := range sides {
		if p.Split == "main" {
			t.Error("side panes should not include main")
		}
	}
}

func TestRenderPrompt(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProjectManagement.Provider = "linear"
	cfg.ProjectManagement.DefaultProject = "PROJ"

	result := cfg.RenderPrompt("Fetch {{provider}} issue {{ticket}} in {{project}} for {{name}}", "PROJ-123", "my-feature")
	expected := "Fetch linear issue PROJ-123 in PROJ for my-feature"
	if result != expected {
		t.Errorf("RenderPrompt:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/projects", filepath.Join(home, "projects")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		got := ExpandHome(tt.input)
		if got != tt.want {
			t.Errorf("ExpandHome(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	cfg := DefaultConfig()
	cfg.Defaults.BranchPrefix = "test"
	cfg.ProjectManagement.Provider = "linear"
	cfg.ProjectManagement.DefaultProject = "TEST"

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Defaults.BranchPrefix != "test" {
		t.Errorf("expected branch_prefix 'test', got %q", loaded.Defaults.BranchPrefix)
	}
	if loaded.ProjectManagement.Provider != "linear" {
		t.Errorf("expected provider 'linear', got %q", loaded.ProjectManagement.Provider)
	}
	if loaded.ProjectManagement.DefaultProject != "TEST" {
		t.Errorf("expected default_project 'TEST', got %q", loaded.ProjectManagement.DefaultProject)
	}
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load on nonexistent should not error: %v", err)
	}
	if cfg.ProjectManagement.Provider != "none" {
		t.Errorf("expected default provider, got %q", cfg.ProjectManagement.Provider)
	}
}

func TestLoadFillsDefaultPrompts(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.ProjectManagement.Prompts.Fetch = ""
	cfg.ProjectManagement.Prompts.Create = ""
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	defaults := DefaultConfig()
	if loaded.ProjectManagement.Prompts.Fetch != defaults.ProjectManagement.Prompts.Fetch {
		t.Error("expected empty fetch prompt to be filled with default")
	}
	if loaded.ProjectManagement.Prompts.Create != defaults.ProjectManagement.Prompts.Create {
		t.Error("expected empty create prompt to be filled with default")
	}
}

func TestConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	expected := filepath.Join(tmpDir, "cwt", "config.toml")
	if got := ConfigPath(); got != expected {
		t.Errorf("ConfigPath: got %q, want %q", got, expected)
	}
}
