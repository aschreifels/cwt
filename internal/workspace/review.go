package workspace

import (
	"fmt"
	"strings"
	"time"

	"github.com/aschreifels/cwt/internal/cmux"
	"github.com/aschreifels/cwt/internal/config"
	"github.com/aschreifels/cwt/internal/git"
)

type ReviewOpts struct {
	PRNumber     int
	Repo         string
	NoCheckout   bool
	SkipPrograms bool
}

type ReviewResult struct {
	WorktreeDir string
	BranchName  string
	WorkspaceID string
	Surfaces    map[string]string
	PRInfo      *git.PRInfo
}

func ReviewWorkspaceName(prNumber int) string {
	return fmt.Sprintf("review-%d", prNumber)
}

func BuildReviewPrompt(cfg config.Config, pr *git.PRInfo) string {
	var b strings.Builder

	b.WriteString(cfg.Review.Prompt)
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("## PR #%d: %s\n", pr.Number, pr.Title))
	b.WriteString(fmt.Sprintf("**Author:** %s\n", pr.Author.Login))
	b.WriteString(fmt.Sprintf("**Branch:** %s → %s\n\n", pr.HeadRefName, pr.BaseRefName))

	if pr.Body != "" {
		b.WriteString("## PR Description\n")
		b.WriteString(pr.Body)
		b.WriteString("\n\n")
	}

	return b.String()
}

func Review(cfg config.Config, opts ReviewOpts, updates chan<- StepUpdate) (*ReviewResult, error) {
	defer close(updates)

	result := &ReviewResult{
		Surfaces: make(map[string]string),
	}

	updates <- StepUpdate{Pane: "pr", Status: "fetching"}
	pr, err := git.GHPRView(opts.PRNumber, opts.Repo)
	if err != nil {
		return nil, fmt.Errorf("fetching pr: %w", err)
	}
	result.PRInfo = pr
	result.BranchName = pr.HeadRefName
	updates <- StepUpdate{Pane: "pr", Status: fmt.Sprintf("#%d: %s", pr.Number, pr.Title), Done: true}

	var worktreeDir string
	if !opts.NoCheckout {
		updates <- StepUpdate{Pane: "worktree", Status: "creating"}
		wsName := ReviewWorkspaceName(opts.PRNumber)
		dir, err := ResolveWorktreeDir(cfg, wsName)
		if err != nil {
			return nil, fmt.Errorf("resolving review worktree dir: %w", err)
		}
		worktreeDir = dir
		result.WorktreeDir = dir

		err = git.WorktreeAddExisting(dir, pr.HeadRefName)
		if err != nil {
			gitErr := git.WorktreeAdd(dir, pr.HeadRefName, pr.BaseRefName)
			if gitErr != nil {
				updates <- StepUpdate{Pane: "worktree", Status: "checkout failed", Err: gitErr}
			} else {
				updates <- StepUpdate{Pane: "worktree", Status: "created", Done: true}
			}
		} else {
			updates <- StepUpdate{Pane: "worktree", Status: "checked out", Done: true}
		}
	}

	mainPane, hasMain := cfg.MainPane()
	if !hasMain {
		return nil, fmt.Errorf("no main pane configured")
	}

	prompt := BuildReviewPrompt(cfg, pr)

	var mainCmd string
	if worktreeDir != "" {
		mainCmd = buildMainCommand(cfg, mainPane, worktreeDir, prompt)
	} else {
		mainCmd = buildMainCommand(cfg, mainPane, ".", prompt)
	}

	updates <- StepUpdate{Pane: "workspace", Status: "creating"}
	ws, err := cmux.NewWorkspace(mainCmd)
	if err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}
	result.WorkspaceID = ws.ID

	wsName := ReviewWorkspaceName(opts.PRNumber)
	if err := cmux.SelectWorkspace(ws.ID); err != nil {
		return nil, fmt.Errorf("selecting workspace: %w", err)
	}
	if err := cmux.RenameWorkspace(ws.ID, wsName); err != nil {
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
		targetDir := worktreeDir
		if targetDir == "" {
			targetDir = "."
		}
		splitResult := &SpawnResult{Surfaces: make(map[string]string)}
		err = createSplits(cfg, ws.ID, targetDir, mainSurface, sidePanes, splitResult, updates)
		if err != nil {
			return result, fmt.Errorf("creating splits: %w", err)
		}
		for k, v := range splitResult.Surfaces {
			result.Surfaces[k] = v
		}
	}

	if !cfg.IsClaude() {
		if prompt != "" {
			updates <- StepUpdate{Pane: mainPane.Name, Status: "waiting for ready"}
			if cmux.WaitForReady(ws.ID, mainSurface, 15*time.Second) {
				time.Sleep(300 * time.Millisecond)
				if err := cmux.SendText(ws.ID, mainSurface, prompt+"\\n"); err != nil {
					updates <- StepUpdate{Pane: mainPane.Name, Status: "prompt send failed", Err: err}
				} else {
					updates <- StepUpdate{Pane: mainPane.Name, Status: "review prompt injected", Done: true}
				}
			} else {
				updates <- StepUpdate{Pane: mainPane.Name, Status: "not ready in time — send prompt manually", Err: fmt.Errorf("timeout")}
			}
		}
	} else if prompt != "" {
		updates <- StepUpdate{Pane: mainPane.Name, Status: "review prompt passed via CLI", Done: true}
	}

	cmux.SetStatus(ws.ID, "pr", fmt.Sprintf("#%d", opts.PRNumber), "git-pull-request", "#8b5cf6")
	cmux.SetStatus(ws.ID, "branch", pr.HeadRefName, "git-branch", "")
	cmux.SetStatus(ws.ID, "base", pr.BaseRefName, "git-merge", "")
	cmux.SetStatus(ws.ID, "author", pr.Author.Login, "code", "")
	cmux.SetStatus(ws.ID, "mode", "review", "eye", "#f59e0b")

	return result, nil
}
