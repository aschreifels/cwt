package workspace

import (
	"strings"
	"testing"

	"github.com/aschreifels/cwt/internal/config"
	"github.com/aschreifels/cwt/internal/git"
)

func TestReviewWorkspaceName(t *testing.T) {
	tests := []struct {
		prNumber int
		want     string
	}{
		{42, "review-42"},
		{1, "review-1"},
		{9999, "review-9999"},
	}

	for _, tt := range tests {
		got := ReviewWorkspaceName(tt.prNumber)
		if got != tt.want {
			t.Errorf("ReviewWorkspaceName(%d): got %q, want %q", tt.prNumber, got, tt.want)
		}
	}
}

func TestBuildReviewPrompt(t *testing.T) {
	cfg := config.DefaultConfig()

	pr := &git.PRInfo{
		Number:      42,
		Title:       "Add review command",
		Body:        "This PR adds a review command to cwt.",
		HeadRefName: "feature/review",
		BaseRefName: "main",
		Author:      git.PRAuthor{Login: "testuser"},
		Additions:   100,
		Deletions:   20,
		Files: []git.PRFile{
			{Path: "cmd/review.go", Additions: 80, Deletions: 0},
			{Path: "internal/workspace/review.go", Additions: 20, Deletions: 20},
		},
	}

	t.Run("includes PR metadata", func(t *testing.T) {
		prompt := BuildReviewPrompt(cfg, pr)
		if !strings.Contains(prompt, "#42") {
			t.Error("expected prompt to contain PR number")
		}
		if !strings.Contains(prompt, "Add review command") {
			t.Error("expected prompt to contain PR title")
		}
		if !strings.Contains(prompt, "testuser") {
			t.Error("expected prompt to contain author")
		}
		if !strings.Contains(prompt, "feature/review") {
			t.Error("expected prompt to contain head branch")
		}
		if !strings.Contains(prompt, "main") {
			t.Error("expected prompt to contain base branch")
		}
	})

	t.Run("includes PR description", func(t *testing.T) {
		prompt := BuildReviewPrompt(cfg, pr)
		if !strings.Contains(prompt, "This PR adds a review command") {
			t.Error("expected prompt to contain PR body")
		}
	})

	t.Run("does not include changed files summary", func(t *testing.T) {
		prompt := BuildReviewPrompt(cfg, pr)
		if strings.Contains(prompt, "Changed Files") {
			t.Error("prompt should not contain changed files summary — agent discovers files itself")
		}
		if strings.Contains(prompt, "+80 / -0") {
			t.Error("prompt should not contain per-file addition/deletion counts")
		}
	})

	t.Run("does not include diff", func(t *testing.T) {
		prompt := BuildReviewPrompt(cfg, pr)
		if strings.Contains(prompt, "## Diff") {
			t.Error("prompt should not contain diff section — agent fetches diff itself")
		}
	})

	t.Run("handles empty body", func(t *testing.T) {
		noBod := *pr
		noBod.Body = ""
		prompt := BuildReviewPrompt(cfg, &noBod)
		if strings.Contains(prompt, "## PR Description") {
			t.Error("should not include description section when body is empty")
		}
	})

	t.Run("includes review config prompt", func(t *testing.T) {
		prompt := BuildReviewPrompt(cfg, pr)
		if !strings.Contains(prompt, "cwt-reviewer") {
			t.Error("expected prompt to reference the review skill")
		}
	})
}
