package graph

import (
	"errors"
	"slices"
	"testing"
)

func TestGraph_AddNode(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B", "C"})

	if !g.HasNode("A") {
		t.Error("node A should exist")
	}

	deps := g.GetDependencies("A")
	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}
}

func TestGraph_RemoveNode(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", nil)
	g.AddNode("B", nil)

	g.RemoveNode("A")

	if g.HasNode("A") {
		t.Error("node A should not exist after removal")
	}
	if !g.HasNode("B") {
		t.Error("node B should still exist")
	}
}

func TestGraph_GetDependents(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"C"})
	g.AddNode("B", []string{"C"})
	g.AddNode("C", nil)

	dependents := g.GetDependents("C")
	if len(dependents) != 2 {
		t.Errorf("expected 2 dependents, got %d", len(dependents))
	}
}

func TestGraph_Validate(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B", "C"})
	g.AddNode("B", nil)

	missing := g.Validate()
	if len(missing) != 1 || missing[0] != "C" {
		t.Errorf("expected missing dependency C, got %v", missing)
	}
}

func TestGraph_Clone(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("B", nil)

	clone := g.Clone()

	if clone.Size() != g.Size() {
		t.Error("clone should have same size")
	}

	g.AddNode("C", nil)
	if clone.Size() == g.Size() {
		t.Error("clone should be independent")
	}
}

func TestGraph_DetectCycles_NoCycle(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("B", []string{"C"})
	g.AddNode("C", nil)

	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("expected no cycles, got %v", cycles)
	}
}

func TestGraph_DetectCycles_SimpleCycle(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("B", []string{"A"})

	cycles := g.DetectCycles()
	if len(cycles) != 1 {
		t.Errorf("expected 1 cycle, got %d", len(cycles))
	}
}

func TestGraph_DetectCycles_SelfCycle(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"A"})

	cycles := g.DetectCycles()
	if len(cycles) != 1 {
		t.Errorf("expected 1 cycle (self-reference), got %d", len(cycles))
	}
}

func TestGraph_DetectCycles_ComplexCycle(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("B", []string{"C"})
	g.AddNode("C", []string{"D"})
	g.AddNode("D", []string{"B"})

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Error("expected at least 1 cycle")
	}
}

func TestGraph_HasCycle(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("B", nil)

	if g.HasCycle() {
		t.Error("should not have cycle")
	}

	g.AddNode("B", []string{"A"})
	if !g.HasCycle() {
		t.Error("should have cycle")
	}
}

func TestGraph_FindCyclePath(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("B", []string{"C"})
	g.AddNode("C", []string{"A"})

	path := g.FindCyclePath("A")
	if len(path) == 0 {
		t.Error("expected cycle path")
	}

	if path[0] != path[len(path)-1] {
		t.Error("cycle path should start and end with same node")
	}
}

func TestGraph_TopologicalSort(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B", "C"})
	g.AddNode("B", []string{"D"})
	g.AddNode("C", []string{"D"})
	g.AddNode("D", nil)

	sorted, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sorted) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(sorted))
	}

	indexOf := func(s []string, v string) int {
		for i, x := range s {
			if x == v {
				return i
			}
		}
		return -1
	}

	if indexOf(sorted, "D") > indexOf(sorted, "B") {
		t.Error("D should come before B")
	}
	if indexOf(sorted, "D") > indexOf(sorted, "C") {
		t.Error("D should come before C")
	}
	if indexOf(sorted, "B") > indexOf(sorted, "A") {
		t.Error("B should come before A")
	}
}

func TestGraph_TopologicalSort_WithCycle(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("B", []string{"A"})

	_, err := g.TopologicalSort()
	if !errors.Is(err, ErrCycleDetected) {
		t.Errorf("expected ErrCycleDetected, got %v", err)
	}
}

func TestGraph_StartupOrder(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("App", []string{"Server", "Database"})
	g.AddNode("Server", []string{"Config"})
	g.AddNode("Database", []string{"Config"})
	g.AddNode("Config", nil)

	order, err := g.StartupOrder()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	indexOf := func(s []string, v string) int {
		for i, x := range s {
			if x == v {
				return i
			}
		}
		return -1
	}

	if indexOf(order, "Config") > indexOf(order, "Server") {
		t.Error("Config should start before Server")
	}
	if indexOf(order, "Config") > indexOf(order, "Database") {
		t.Error("Config should start before Database")
	}
	if indexOf(order, "Server") > indexOf(order, "App") {
		t.Error("Server should start before App")
	}
}

func TestGraph_ShutdownOrder(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("App", []string{"Server"})
	g.AddNode("Server", []string{"Database"})
	g.AddNode("Database", nil)

	order, err := g.ShutdownOrder()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	indexOf := func(s []string, v string) int {
		for i, x := range s {
			if x == v {
				return i
			}
		}
		return -1
	}

	if indexOf(order, "App") > indexOf(order, "Server") {
		t.Error("App should shutdown before Server")
	}
	if indexOf(order, "Server") > indexOf(order, "Database") {
		t.Error("Server should shutdown before Database")
	}
}

func TestGraph_ResolutionOrder(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("A", []string{"B", "C"})
	g.AddNode("B", []string{"D"})
	g.AddNode("C", nil)
	g.AddNode("D", nil)

	order, err := g.ResolutionOrder("A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order[len(order)-1] != "A" {
		t.Error("target should be last in resolution order")
	}

	if !slices.Contains(order, "B") || !slices.Contains(order, "C") || !slices.Contains(order, "D") {
		t.Error("all dependencies should be in resolution order")
	}
}

func TestGraph_ParallelStartupGroups(t *testing.T) {
	t.Parallel()

	g := New()
	g.AddNode("App", []string{"Server", "Worker"})
	g.AddNode("Server", []string{"Database", "Cache"})
	g.AddNode("Worker", []string{"Database"})
	g.AddNode("Database", []string{"Config"})
	g.AddNode("Cache", []string{"Config"})
	g.AddNode("Config", nil)

	groups, err := g.ParallelStartupGroups()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(groups) == 0 {
		t.Fatal("expected at least one group")
	}

	if !slices.Contains(groups[0].Nodes, "Config") {
		t.Error("Config should be in first group (level 0)")
	}
}

func BenchmarkGraph_DetectCycles(b *testing.B) {
	g := New()
	for i := 0; i < 100; i++ {
		var deps []string
		if i > 0 {
			deps = []string{string(rune('A' + i - 1))}
		}
		g.AddNode(string(rune('A'+i)), deps)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g.DetectCycles()
	}
}

func BenchmarkGraph_TopologicalSort(b *testing.B) {
	g := New()
	for i := 0; i < 100; i++ {
		var deps []string
		if i > 0 {
			deps = []string{string(rune('A' + i - 1))}
		}
		g.AddNode(string(rune('A'+i)), deps)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = g.TopologicalSort()
	}
}
