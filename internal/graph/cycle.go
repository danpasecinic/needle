package graph

type CycleDetector struct {
	graph   *Graph
	index   int
	stack   []string
	onStack map[string]bool
	indices map[string]int
	lowlink map[string]int
	sccs    [][]string
}

func (g *Graph) DetectCycles() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	detector := &CycleDetector{
		graph:   g,
		index:   0,
		stack:   make([]string, 0),
		onStack: make(map[string]bool),
		indices: make(map[string]int),
		lowlink: make(map[string]int),
		sccs:    make([][]string, 0),
	}

	for id := range g.nodes {
		if _, visited := detector.indices[id]; !visited {
			detector.strongConnect(id)
		}
	}

	var cycles [][]string
	for _, scc := range detector.sccs {
		if len(scc) > 1 {
			cycles = append(cycles, scc)
		} else if len(scc) == 1 {
			id := scc[0]
			for _, dep := range g.edges[id] {
				if dep == id {
					cycles = append(cycles, scc)
					break
				}
			}
		}
	}

	return cycles
}

func (d *CycleDetector) strongConnect(id string) {
	d.indices[id] = d.index
	d.lowlink[id] = d.index
	d.index++
	d.stack = append(d.stack, id)
	d.onStack[id] = true

	for _, dep := range d.graph.edges[id] {
		if _, exists := d.graph.nodes[dep]; !exists {
			continue
		}

		if _, visited := d.indices[dep]; !visited {
			d.strongConnect(dep)
			d.lowlink[id] = min(d.lowlink[id], d.lowlink[dep])
		} else if d.onStack[dep] {
			d.lowlink[id] = min(d.lowlink[id], d.indices[dep])
		}
	}

	if d.lowlink[id] == d.indices[id] {
		var scc []string
		for {
			n := len(d.stack) - 1
			w := d.stack[n]
			d.stack = d.stack[:n]
			d.onStack[w] = false
			scc = append(scc, w)
			if w == id {
				break
			}
		}
		d.sccs = append(d.sccs, scc)
	}
}

func (g *Graph) HasCycle() bool {
	g.mu.RLock()
	if g.cycleValid {
		result := g.hasCycle
		g.mu.RUnlock()
		return result
	}
	g.mu.RUnlock()

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.cycleValid {
		return g.hasCycle
	}

	g.hasCycle = g.hasCycleUnsafe()
	g.cycleValid = true
	return g.hasCycle
}

func (g *Graph) hasCycleUnsafe() bool {
	white := make(map[string]bool, len(g.nodes))
	gray := make(map[string]bool, len(g.nodes))

	for id := range g.nodes {
		white[id] = true
	}

	var hasCycle bool
	var dfs func(id string) bool
	dfs = func(id string) bool {
		white[id] = false
		gray[id] = true

		for _, dep := range g.edges[id] {
			if _, exists := g.nodes[dep]; !exists {
				continue
			}
			if gray[dep] {
				return true
			}
			if white[dep] && dfs(dep) {
				return true
			}
		}

		gray[id] = false
		return false
	}

	for id := range g.nodes {
		if white[id] {
			if dfs(id) {
				hasCycle = true
				break
			}
		}
	}

	return hasCycle
}

func (g *Graph) FindCyclePath(start string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	path := make([]string, 0)
	inPath := make(map[string]bool)

	var dfs func(id string) []string
	dfs = func(id string) []string {
		if inPath[id] {
			cyclePath := make([]string, 0)
			found := false
			for _, p := range path {
				if p == id {
					found = true
				}
				if found {
					cyclePath = append(cyclePath, p)
				}
			}
			cyclePath = append(cyclePath, id)
			return cyclePath
		}

		if visited[id] {
			return nil
		}

		visited[id] = true
		path = append(path, id)
		inPath[id] = true

		for _, dep := range g.edges[id] {
			if _, exists := g.nodes[dep]; !exists {
				continue
			}
			if cycle := dfs(dep); cycle != nil {
				return cycle
			}
		}

		path = path[:len(path)-1]
		inPath[id] = false
		return nil
	}

	return dfs(start)
}

func (g *Graph) GetAllCyclePaths() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		return nil
	}

	var allPaths [][]string
	for _, scc := range cycles {
		if len(scc) > 0 {
			path := g.FindCyclePath(scc[0])
			if path != nil {
				allPaths = append(allPaths, path)
			}
		}
	}

	return allPaths
}
