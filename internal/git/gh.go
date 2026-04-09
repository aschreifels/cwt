package git

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type PRInfo struct {
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	HeadRefName string   `json:"headRefName"`
	BaseRefName string   `json:"baseRefName"`
	State       string   `json:"state"`
	Author      PRAuthor `json:"author"`
	Additions   int      `json:"additions"`
	Deletions   int      `json:"deletions"`
	Files       []PRFile `json:"files"`
}

type PRAuthor struct {
	Login string `json:"login"`
}

type PRFile struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

func GHPRView(prNumber int, repo string) (*PRInfo, error) {
	args := []string{"pr", "view", fmt.Sprintf("%d", prNumber),
		"--json", "number,title,body,headRefName,baseRefName,state,author,additions,deletions,files"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}

	cmd := exec.Command("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh pr view: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh pr view: %w", err)
	}

	var info PRInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("parsing pr json: %w", err)
	}
	return &info, nil
}

func GHPRDiff(prNumber int, repo string) (string, error) {
	args := []string{"pr", "diff", fmt.Sprintf("%d", prNumber)}
	if repo != "" {
		args = append(args, "--repo", repo)
	}

	cmd := exec.Command("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gh pr diff: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("gh pr diff: %w", err)
	}
	return string(out), nil
}

func GHRepoFromRemote() (string, error) {
	cmd := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gh repo view: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("gh repo view: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func ParsePRFromURL(url string) (repo string, prNumber int, err error) {
	url = strings.TrimRight(url, "/")

	// https://github.com/owner/repo/pull/123
	parts := strings.Split(url, "/")
	if len(parts) < 5 {
		return "", 0, fmt.Errorf("invalid PR URL: %s", url)
	}

	pullIdx := -1
	for i, p := range parts {
		if p == "pull" {
			pullIdx = i
			break
		}
	}

	if pullIdx < 0 || pullIdx+1 >= len(parts) {
		return "", 0, fmt.Errorf("no pull request number found in URL: %s", url)
	}

	n := 0
	for _, c := range parts[pullIdx+1] {
		if c < '0' || c > '9' {
			return "", 0, fmt.Errorf("invalid PR number in URL: %s", url)
		}
		n = n*10 + int(c-'0')
	}

	ownerIdx := pullIdx - 2
	repoIdx := pullIdx - 1
	if ownerIdx < 0 || repoIdx < 0 {
		return "", 0, fmt.Errorf("cannot determine repo from URL: %s", url)
	}

	return parts[ownerIdx] + "/" + parts[repoIdx], n, nil
}
