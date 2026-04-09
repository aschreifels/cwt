package skills

import _ "embed"

//go:embed cmux-notifications/SKILL.md
var CmuxNotifications []byte

//go:embed cwt-orchestrator/SKILL.md
var CwtOrchestrator []byte

//go:embed cwt-reviewer/SKILL.md
var CwtReviewer []byte

//go:embed cwt-reviewer-comments/SKILL.md
var CwtReviewerComments []byte

//go:embed cwt-reviewer-go/SKILL.md
var CwtReviewerGo []byte

//go:embed cwt-reviewer-typescript/SKILL.md
var CwtReviewerTypescript []byte

//go:embed cwt-reviewer-database/SKILL.md
var CwtReviewerDatabase []byte

//go:embed cwt-reviewer-infra/SKILL.md
var CwtReviewerInfra []byte

type Skill struct {
	Name        string
	Description string
	Dir         string
	Content     []byte
}

func All() []Skill {
	return []Skill{
		{
			Name:        "cmux-notifications",
			Description: "Teaches your AI agent to use cmux sidebar APIs (status, progress, log, notify)",
			Dir:         "cmux-notifications",
			Content:     CmuxNotifications,
		},
		{
			Name:        "cwt-orchestrator",
			Description: "Teaches your AI agent to analyze projects and spawn parallel worktrees via cwt",
			Dir:         "cwt-orchestrator",
			Content:     CwtOrchestrator,
		},
		{
			Name:        "cwt-reviewer",
			Description: "Routes PR reviews to language/domain-specific review skills based on file detection",
			Dir:         "cwt-reviewer",
			Content:     CwtReviewer,
		},
		{
			Name:        "cwt-reviewer-comments",
			Description: "Posts review findings as inline GitHub PR comments via the gh CLI",
			Dir:         "cwt-reviewer-comments",
			Content:     CwtReviewerComments,
		},
		{
			Name:        "cwt-reviewer-go",
			Description: "Go-specific PR review checklist: build, test, lint, idiomatic patterns, security",
			Dir:         "cwt-reviewer-go",
			Content:     CwtReviewerGo,
		},
		{
			Name:        "cwt-reviewer-typescript",
			Description: "TypeScript-specific PR review checklist: types, async, React, security, performance",
			Dir:         "cwt-reviewer-typescript",
			Content:     CwtReviewerTypescript,
		},
		{
			Name:        "cwt-reviewer-database",
			Description: "Database PR review checklist: migration safety, indexes, query performance, compatibility",
			Dir:         "cwt-reviewer-database",
			Content:     CwtReviewerDatabase,
		},
		{
			Name:        "cwt-reviewer-infra",
			Description: "Infrastructure PR review checklist: Kubernetes, Terraform, Docker, CI/CD, scripts",
			Dir:         "cwt-reviewer-infra",
			Content:     CwtReviewerInfra,
		},
	}
}

func ReviewSkills() []Skill {
	all := All()
	var review []Skill
	for _, s := range all {
		if s.Dir == "cwt-reviewer" ||
			s.Dir == "cwt-reviewer-comments" ||
			s.Dir == "cwt-reviewer-go" ||
			s.Dir == "cwt-reviewer-typescript" ||
			s.Dir == "cwt-reviewer-database" ||
			s.Dir == "cwt-reviewer-infra" {
			review = append(review, s)
		}
	}
	return review
}
