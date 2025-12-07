package graph

import "errors"

var ErrCycleDetected = errors.New("cycle detected in graph")

func (g *Graph) TopologicalSort() ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodeCount := len(g.nodes)
	dependents := make(map[string][]string, nodeCount)
	inDegree := make(map[string]int, nodeCount)

	for id := range g.nodes {
		inDegree[id] = 0
	}

	for id, deps := range g.edges {
		for _, dep := range deps {
			if _, exists := g.nodes[dep]; exists {
				dependents[dep] = append(dependents[dep], id)
				inDegree[id]++
			}
		}
	}

	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted = append(sorted, node)

		for _, dependent := range dependents[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(sorted) != len(g.nodes) {
		return nil, ErrCycleDetected
	}

	return sorted, nil
}

func (g *Graph) ReverseTopologicalSort() ([]string, error) {
	sorted, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}

	n := len(sorted)
	reversed := make([]string, n)
	for i, v := range sorted {
		reversed[n-1-i] = v
	}

	return reversed, nil
}

func (g *Graph) StartupOrder() ([]string, error) {
	return g.TopologicalSort()
}

func (g *Graph) ShutdownOrder() ([]string, error) {
	return g.ReverseTopologicalSort()
}

func (g *Graph) ResolutionOrder(target string) ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.nodes[target]; !exists {
		return []string{target}, nil
	}

	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var order []string

	var visit func(id string) error
	visit = func(id string) error {
		if visiting[id] {
			return ErrCycleDetected
		}
		if visited[id] {
			return nil
		}

		visiting[id] = true

		for _, dep := range g.edges[id] {
			if _, exists := g.nodes[dep]; !exists {
				continue
			}
			if err := visit(dep); err != nil {
				return err
			}
		}

		visiting[id] = false
		visited[id] = true
		order = append(order, id)
		return nil
	}

	if err := visit(target); err != nil {
		return nil, err
	}

	return order, nil
}

type ParallelGroup struct {
	Level int
	Nodes []string
}

func (g *Graph) ParallelShutdownGroups() ([]ParallelGroup, error) {
	groups, err := g.ParallelStartupGroups()
	if err != nil {
		return nil, err
	}

	n := len(groups)
	reversed := make([]ParallelGroup, n)
	for i, group := range groups {
		reversed[n-1-i] = ParallelGroup{
			Level: n - 1 - i,
			Nodes: group.Nodes,
		}
	}

	return reversed, nil
}

func (g *Graph) ParallelStartupGroups() ([]ParallelGroup, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	levels := make(map[string]int, len(g.nodes))

	var calculateLevel func(id string) int
	calculateLevel = func(id string) int {
		if level, ok := levels[id]; ok {
			return level
		}

		deps := g.edges[id]
		if len(deps) == 0 {
			levels[id] = 0
			return 0
		}

		maxDepLevel := -1
		for _, dep := range deps {
			if _, exists := g.nodes[dep]; !exists {
				continue
			}
			depLevel := calculateLevel(dep)
			if depLevel > maxDepLevel {
				maxDepLevel = depLevel
			}
		}

		level := maxDepLevel + 1
		levels[id] = level
		return level
	}

	for id := range g.nodes {
		calculateLevel(id)
	}

	groupMap := make(map[int][]string)
	maxLevel := 0
	for id, level := range levels {
		groupMap[level] = append(groupMap[level], id)
		if level > maxLevel {
			maxLevel = level
		}
	}

	groups := make([]ParallelGroup, 0, maxLevel+1)
	for level := 0; level <= maxLevel; level++ {
		if nodes, ok := groupMap[level]; ok {
			groups = append(
				groups, ParallelGroup{
					Level: level,
					Nodes: nodes,
				},
			)
		}
	}

	return groups, nil
}
