package skills

import (
	"testing"
)

func TestAllSkills(t *testing.T) {
	all := All()

	if len(all) != 8 {
		t.Fatalf("expected 8 skills, got %d", len(all))
	}

	expected := map[string]bool{
		"cmux-notifications":      false,
		"cwt-orchestrator":        false,
		"cwt-reviewer":            false,
		"cwt-reviewer-comments":   false,
		"cwt-reviewer-go":         false,
		"cwt-reviewer-typescript": false,
		"cwt-reviewer-database":   false,
		"cwt-reviewer-infra":      false,
	}

	for _, s := range all {
		if _, ok := expected[s.Name]; !ok {
			t.Errorf("unexpected skill: %s", s.Name)
		}
		expected[s.Name] = true

		if s.Description == "" {
			t.Errorf("skill %s has empty description", s.Name)
		}
		if s.Dir == "" {
			t.Errorf("skill %s has empty dir", s.Name)
		}
		if len(s.Content) == 0 {
			t.Errorf("skill %s has empty content", s.Name)
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing skill: %s", name)
		}
	}
}

func TestSkillContentNotEmpty(t *testing.T) {
	if len(CmuxNotifications) == 0 {
		t.Error("CmuxNotifications embed is empty")
	}
	if len(CwtOrchestrator) == 0 {
		t.Error("CwtOrchestrator embed is empty")
	}
	if len(CwtReviewer) == 0 {
		t.Error("CwtReviewer embed is empty")
	}
	if len(CwtReviewerComments) == 0 {
		t.Error("CwtReviewerComments embed is empty")
	}
	if len(CwtReviewerGo) == 0 {
		t.Error("CwtReviewerGo embed is empty")
	}
	if len(CwtReviewerTypescript) == 0 {
		t.Error("CwtReviewerTypescript embed is empty")
	}
	if len(CwtReviewerDatabase) == 0 {
		t.Error("CwtReviewerDatabase embed is empty")
	}
	if len(CwtReviewerInfra) == 0 {
		t.Error("CwtReviewerInfra embed is empty")
	}
}

func TestReviewSkills(t *testing.T) {
	review := ReviewSkills()
	if len(review) != 6 {
		t.Fatalf("expected 6 review skills, got %d", len(review))
	}

	names := make(map[string]bool)
	for _, s := range review {
		names[s.Name] = true
	}

	expected := []string{
		"cwt-reviewer",
		"cwt-reviewer-comments",
		"cwt-reviewer-go",
		"cwt-reviewer-typescript",
		"cwt-reviewer-database",
		"cwt-reviewer-infra",
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("ReviewSkills missing: %s", name)
		}
	}
}
