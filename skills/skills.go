package skills

import _ "embed"

//go:embed cmux-notifications/SKILL.md
var CmuxNotifications []byte

//go:embed cwt-orchestrator/SKILL.md
var CwtOrchestrator []byte

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
			Description: "Teaches Crush to use cmux sidebar APIs (status, progress, log, notify)",
			Dir:         "cmux-notifications",
			Content:     CmuxNotifications,
		},
		{
			Name:        "cwt-orchestrator",
			Description: "Teaches Crush to analyze projects and spawn parallel worktrees via cwt",
			Dir:         "cwt-orchestrator",
			Content:     CwtOrchestrator,
		},
	}
}
