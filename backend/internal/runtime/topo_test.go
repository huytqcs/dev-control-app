package runtime

import "testing"

func indexOf(order []string, id string) int {
	for i, v := range order {
		if v == id {
			return i
		}
	}
	return -1
}

func TestTopoSortServices_RespectsDependencies(t *testing.T) {
	ids := []string{"app", "be", "db"}
	deps := map[string][]string{
		"app": {"be"},
		"be":  {"db"},
		"db":  {},
	}

	order, hadCycle := topoSortServices(ids, deps)
	if hadCycle {
		t.Fatalf("expected no cycle")
	}
	if indexOf(order, "db") > indexOf(order, "be") {
		t.Fatalf("db must come before be, got %v", order)
	}
	if indexOf(order, "be") > indexOf(order, "app") {
		t.Fatalf("be must come before app, got %v", order)
	}
}

func TestTopoSortServices_IgnoresDependenciesOutsideSet(t *testing.T) {
	ids := []string{"app"}
	deps := map[string][]string{"app": {"not-in-preset"}}

	order, hadCycle := topoSortServices(ids, deps)
	if hadCycle {
		t.Fatalf("expected no cycle")
	}
	if len(order) != 1 || order[0] != "app" {
		t.Fatalf("expected [app], got %v", order)
	}
}

func TestTopoSortServices_CycleFallsBackToConfigOrder(t *testing.T) {
	ids := []string{"a", "b"}
	deps := map[string][]string{
		"a": {"b"},
		"b": {"a"},
	}

	order, hadCycle := topoSortServices(ids, deps)
	if !hadCycle {
		t.Fatalf("expected cycle to be detected")
	}
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("expected fallback to original order [a b], got %v", order)
	}
}

func TestTopoSortServices_EmptyInput(t *testing.T) {
	order, hadCycle := topoSortServices(nil, nil)
	if hadCycle {
		t.Fatalf("expected no cycle for empty input")
	}
	if len(order) != 0 {
		t.Fatalf("expected empty order, got %v", order)
	}
}
