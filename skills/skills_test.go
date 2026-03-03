package skills

import (
	"testing"
)

func TestAllSkills(t *testing.T) {
	all := All()

	if len(all) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(all))
	}

	expected := map[string]bool{
		"cmux-notifications": false,
		"cwt-orchestrator":   false,
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
}
