package graph

import "sync"

type Node struct {
	ID           string
	Dependencies []string
}

type Graph struct {
	mu         sync.RWMutex
	nodes      map[string]*Node
	edges      map[string][]string
	cycleValid bool
	hasCycle   bool
}

func New() *Graph {
	return &Graph{
		nodes: make(map[string]*Node),
		edges: make(map[string][]string),
	}
}

func (g *Graph) AddNode(id string, dependencies []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodes[id] = &Node{
		ID:           id,
		Dependencies: dependencies,
	}
	g.edges[id] = dependencies
	g.cycleValid = false
}

func (g *Graph) RemoveNode(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.nodes, id)
	delete(g.edges, id)
	g.cycleValid = false
}

func (g *Graph) HasNode(id string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	_, exists := g.nodes[id]
	return exists
}

func (g *Graph) GetNode(id string) (*Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, exists := g.nodes[id]
	if !exists {
		return nil, false
	}

	nodeCopy := &Node{
		ID:           node.ID,
		Dependencies: make([]string, len(node.Dependencies)),
	}
	copy(nodeCopy.Dependencies, node.Dependencies)
	return nodeCopy, true
}

func (g *Graph) GetDependencies(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	deps, exists := g.edges[id]
	if !exists {
		return nil
	}

	result := make([]string, len(deps))
	copy(result, deps)
	return result
}

func (g *Graph) GetDependents(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var dependents []string
	for nodeID, deps := range g.edges {
		for _, dep := range deps {
			if dep == id {
				dependents = append(dependents, nodeID)
				break
			}
		}
	}
	return dependents
}

func (g *Graph) Nodes() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodes := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		nodes = append(nodes, id)
	}
	return nodes
}

func (g *Graph) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.nodes)
}

func (g *Graph) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodes = make(map[string]*Node)
	g.edges = make(map[string][]string)
	g.cycleValid = false
}

func (g *Graph) Clone() *Graph {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clone := New()
	for id, node := range g.nodes {
		deps := make([]string, len(node.Dependencies))
		copy(deps, node.Dependencies)
		clone.nodes[id] = &Node{
			ID:           node.ID,
			Dependencies: deps,
		}
		clone.edges[id] = deps
	}
	return clone
}

func (g *Graph) Validate() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var missing []string
	seen := make(map[string]bool)

	for _, deps := range g.edges {
		for _, dep := range deps {
			if _, exists := g.nodes[dep]; !exists && !seen[dep] {
				missing = append(missing, dep)
				seen[dep] = true
			}
		}
	}

	return missing
}
