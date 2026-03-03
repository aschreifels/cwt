package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aschreifels/cwt/internal/cmux"
	"github.com/aschreifels/cwt/internal/config"
	"github.com/aschreifels/cwt/internal/git"
)

type SpawnOpts struct {
	Name         string
	BaseBranch   string
	Existing     bool
	ExistBranch  string
	Ticket       string
	CreateDraft  bool
	SkipPrograms bool
}

type SpawnResult struct {
	WorktreeDir string
	BranchName  string
	WorkspaceID string
	Surfaces    map[string]string
}

type StepUpdate struct {
	Pane   string
	Status string
	Done   bool
	Err    error
}

func BuildBranchName(cfg config.Config, opts SpawnOpts) string {
	if opts.Existing {
		if opts.ExistBranch != "" {
			return opts.ExistBranch
		}
		return opts.Name
	}

	var parts []string
	if cfg.Defaults.BranchPrefix != "" {
		parts = append(parts, cfg.Defaults.BranchPrefix)
	}

	name := opts.Name
	if opts.Ticket != "" {
		name = opts.Ticket + "_" + name
	}

	parts = append(parts, name)

	if cfg.Defaults.BranchPrefix != "" {
		return parts[0] + "/" + strings.Join(parts[1:], "/")
	}
	return strings.Join(parts, "/")
}

func ResolveBaseBranch(cfg config.Config, opts SpawnOpts) string {
	if opts.BaseBranch != "" {
		return opts.BaseBranch
	}
	if cfg.Defaults.BaseBranch != "" {
		return cfg.Defaults.BaseBranch
	}
	return git.DefaultBranch()
}

func ResolveWorktreeDir(cfg config.Config, name string) (string, error) {
	if cfg.Defaults.WorktreeDir != "" {
		return filepath.Join(config.ExpandHome(cfg.Defaults.WorktreeDir), name), nil
	}
	base, err := git.DefaultWorktreeBase()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, name), nil
}

func Spawn(cfg config.Config, opts SpawnOpts, updates chan<- StepUpdate) (*SpawnResult, error) {
	defer close(updates)

	result := &SpawnResult{
		Surfaces: make(map[string]string),
	}

	branchName := BuildBranchName(cfg, opts)
	baseBranch := ResolveBaseBranch(cfg, opts)
	result.BranchName = branchName

	worktreeDir, err := ResolveWorktreeDir(cfg, opts.Name)
	if err != nil {
		return nil, fmt.Errorf("resolving worktree dir: %w", err)
	}
	result.WorktreeDir = worktreeDir

	updates <- StepUpdate{Pane: "worktree", Status: "creating"}
	if _, err := os.Stat(worktreeDir); err == nil {
		updates <- StepUpdate{Pane: "worktree", Status: "reusing existing", Done: true}
	} else {
		if err := os.MkdirAll(filepath.Dir(worktreeDir), 0o755); err != nil {
			return nil, fmt.Errorf("creating worktree parent: %w", err)
		}

		if opts.Existing {
			err = git.WorktreeAddExisting(worktreeDir, branchName)
		} else {
			err = git.WorktreeAdd(worktreeDir, branchName, baseBranch)
		}
		if err != nil {
			return nil, fmt.Errorf("creating worktree: %w", err)
		}
		updates <- StepUpdate{Pane: "worktree", Status: "created", Done: true}
	}

	mainPane, hasMain := cfg.MainPane()
	if !hasMain {
		return nil, fmt.Errorf("no main pane configured")
	}

	mainCmd := expandCommand(mainPane.Command, worktreeDir)
	updates <- StepUpdate{Pane: "workspace", Status: "creating"}

	ws, err := cmux.NewWorkspace(mainCmd)
	if err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}
	result.WorkspaceID = ws.ID

	if err := cmux.SelectWorkspace(ws.ID); err != nil {
		return nil, fmt.Errorf("selecting workspace: %w", err)
	}
	if err := cmux.RenameWorkspace(ws.ID, opts.Name); err != nil {
		return nil, fmt.Errorf("renaming workspace: %w", err)
	}

	time.Sleep(300 * time.Millisecond)

	surfaces, err := cmux.ListPaneSurfaces(ws.ID)
	if err != nil {
		return nil, fmt.Errorf("finding main surface: %w", err)
	}
	if len(surfaces) == 0 {
		return nil, fmt.Errorf("no surfaces found in workspace")
	}
	mainSurface := surfaces[0]
	result.Surfaces[mainPane.Name] = mainSurface

	updates <- StepUpdate{Pane: "workspace", Status: "created", Done: true}

	if !opts.SkipPrograms {
		sidePanes := cfg.SidePanes()
		err = createSplits(cfg, ws.ID, worktreeDir, mainSurface, sidePanes, result, updates)
		if err != nil {
			return result, fmt.Errorf("creating splits: %w", err)
		}
	}

	prompt := resolvePrompt(cfg, opts)
	if prompt != "" {
		updates <- StepUpdate{Pane: mainPane.Name, Status: "waiting for ready"}
		if cmux.WaitForReady(ws.ID, mainSurface, 15*time.Second) {
			time.Sleep(300 * time.Millisecond)
			if err := cmux.SendText(ws.ID, mainSurface, prompt+"\\n"); err != nil {
				updates <- StepUpdate{Pane: mainPane.Name, Status: "prompt send failed", Err: err}
			} else {
				updates <- StepUpdate{Pane: mainPane.Name, Status: "prompt injected", Done: true}
			}
		} else {
			updates <- StepUpdate{Pane: mainPane.Name, Status: "not ready in time — send prompt manually", Err: fmt.Errorf("timeout")}
		}
	}

	cmux.SetStatus(ws.ID, "branch", branchName, "git-branch", "")
	cmux.SetStatus(ws.ID, "base", baseBranch, "git-merge", "")
	if opts.Ticket != "" {
		cmux.SetStatus(ws.ID, "ticket", opts.Ticket, "lightning", "#5e6ad2")
	}
	if cfg.HasProjectManagement() {
		cmux.SetStatus(ws.ID, "provider", cfg.ProjectManagement.Provider, "gear", "")
	}
	if opts.CreateDraft {
		cmux.SetStatus(ws.ID, "ticket", "draft", "note", "#f59e0b")
	}

	return result, nil
}

func resolvePrompt(cfg config.Config, opts SpawnOpts) string {
	if !cfg.HasProjectManagement() {
		return ""
	}

	if opts.Ticket != "" {
		return cfg.RenderPrompt(cfg.ProjectManagement.Prompts.Fetch, opts.Ticket, opts.Name)
	}

	if opts.CreateDraft {
		return cfg.RenderPrompt(cfg.ProjectManagement.Prompts.Create, "", opts.Name)
	}

	return ""
}

func createSplits(cfg config.Config, wsID, worktreeDir, mainSurface string, sidePanes []config.PaneConfig, result *SpawnResult, updates chan<- StepUpdate) error {
	if len(sidePanes) == 0 {
		return nil
	}

	updates <- StepUpdate{Pane: sidePanes[0].Name, Status: "starting"}
	rightSurface, err := cmux.NewSplit("right", wsID)
	if err != nil {
		return fmt.Errorf("right split: %w", err)
	}
	time.Sleep(300 * time.Millisecond)

	if len(sidePanes) == 1 {
		result.Surfaces[sidePanes[0].Name] = rightSurface
		launchInPane(wsID, rightSurface, sidePanes[0], worktreeDir)
		updates <- StepUpdate{Pane: sidePanes[0].Name, Status: "ready", Done: true}
		return nil
	}

	result.Surfaces[sidePanes[0].Name] = rightSurface
	launchInPane(wsID, rightSurface, sidePanes[0], worktreeDir)
	updates <- StepUpdate{Pane: sidePanes[0].Name, Status: "ready", Done: true}

	lastSurface := rightSurface
	for i := 1; i < len(sidePanes); i++ {
		pane := sidePanes[i]
		updates <- StepUpdate{Pane: pane.Name, Status: "starting"}

		surface, err := cmux.NewSplitOnPanel("down", wsID, lastSurface)
		if err != nil {
			updates <- StepUpdate{Pane: pane.Name, Status: "failed", Err: err}
			continue
		}

		result.Surfaces[pane.Name] = surface
		launchInPane(wsID, surface, pane, worktreeDir)
		updates <- StepUpdate{Pane: pane.Name, Status: "ready", Done: true}
		lastSurface = surface
	}

	return nil
}

func launchInPane(wsID, surfaceRef string, pane config.PaneConfig, worktreeDir string) {
	cmd := fmt.Sprintf("cd %s && %s", shellQuote(worktreeDir), expandCommand(pane.Command, worktreeDir))
	cmux.SendPanel(wsID, surfaceRef, cmd)
	cmux.SendKeyPanel(wsID, surfaceRef, "enter")
}

func expandCommand(cmd, worktreeDir string) string {
	return strings.ReplaceAll(cmd, "{{worktree_dir}}", worktreeDir)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
