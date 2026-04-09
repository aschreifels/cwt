package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func RepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

func DefaultBranch() string {
	for _, name := range []string{"main", "master"} {
		cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+name)
		if cmd.Run() == nil {
			return name
		}
	}
	return "main"
}

func WorktreeAdd(worktreeDir, branchName, baseBranch string) error {
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir, baseBranch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add: %w: %s", err, out)
	}
	return nil
}

func WorktreeAddExisting(worktreeDir, branchName string) error {
	cmd := exec.Command("git", "worktree", "add", worktreeDir, branchName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add: %w: %s", err, out)
	}
	return nil
}

func WorktreeRemove(worktreeDir string) error {
	cmd := exec.Command("git", "worktree", "remove", worktreeDir, "--force")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove: %w: %s", err, out)
	}
	return nil
}

func WorktreeRemoveFast(worktreeDir string) error {
	tmpDir := worktreeDir + fmt.Sprintf(".cwt-rm-%d", os.Getpid())

	renamed := false
	if err := os.Rename(worktreeDir, tmpDir); err == nil {
		renamed = true
	} else {
		tmpDir = worktreeDir
	}

	cmd := exec.Command("git", "worktree", "prune")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree prune: %w: %s", err, out)
	}

	if renamed {
		rmCmd := exec.Command("/bin/rm", "-rf", tmpDir)
		rmCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := rmCmd.Start(); err != nil {
			os.RemoveAll(tmpDir)
			return nil
		}
		go func() {
			rmCmd.Wait()
		}()
	} else {
		rmCmd := exec.Command("/bin/rm", "-rf", tmpDir)
		if out, err := rmCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("removing worktree directory: %w: %s", err, out)
		}
	}

	return nil
}

func BranchFromWorktree(worktreeDir string) (string, error) {
	cmd := exec.Command("git", "-C", worktreeDir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("detecting branch: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("could not detect branch name")
	}
	return branch, nil
}

func DeleteBranch(name string) error {
	cmd := exec.Command("git", "branch", "-D", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch -D: %w: %s", err, out)
	}
	return nil
}

func WorktreeList() ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	var worktrees []WorktreeInfo
	var current WorktreeInfo

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = WorktreeInfo{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			current.Bare = true
		}
	}
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

func DefaultWorktreeBase() (string, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(root), "worktrees"), nil
}

type WorktreeInfo struct {
	Path   string
	Branch string
	Bare   bool
}
