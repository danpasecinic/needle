package needle

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type GraphInfo struct {
	Services []ServiceInfo
}

type ServiceInfo struct {
	Key          string
	Dependencies []string
	Dependents   []string
	Instantiated bool
	Scope        string
}

func (c *Container) Graph() GraphInfo {
	keys := c.internal.Keys()
	sort.Strings(keys)

	graph := c.internal.Graph()
	services := make([]ServiceInfo, 0, len(keys))

	for _, key := range keys {
		deps := graph.GetDependencies(key)
		dependents := graph.GetDependents(key)
		_, instantiated := c.internal.GetInstance(key)

		services = append(
			services, ServiceInfo{
				Key:          key,
				Dependencies: deps,
				Dependents:   dependents,
				Instantiated: instantiated,
			},
		)
	}

	return GraphInfo{Services: services}
}

func (c *Container) PrintGraph() {
	c.FprintGraph(os.Stdout)
}

func (c *Container) FprintGraph(w io.Writer) {
	info := c.Graph()

	if len(info.Services) == 0 {
		_, _ = fmt.Fprintln(w, "(empty container)")
		return
	}

	for _, svc := range info.Services {
		status := "○"
		if svc.Instantiated {
			status = "●"
		}

		if len(svc.Dependencies) == 0 {
			_, _ = fmt.Fprintf(w, "%s %s\n", status, svc.Key)
		} else {
			_, _ = fmt.Fprintf(w, "%s %s ← %s\n", status, svc.Key, strings.Join(svc.Dependencies, ", "))
		}
	}
}

func (c *Container) SprintGraph() string {
	var sb strings.Builder
	c.FprintGraph(&sb)
	return sb.String()
}

func (c *Container) PrintGraphDOT() {
	c.FprintGraphDOT(os.Stdout)
}

func (c *Container) FprintGraphDOT(w io.Writer) {
	info := c.Graph()

	_, _ = fmt.Fprintln(w, "digraph dependencies {")
	_, _ = fmt.Fprintln(w, "  rankdir=LR;")
	_, _ = fmt.Fprintln(w, "  node [shape=box];")

	for _, svc := range info.Services {
		label := escapeLabel(svc.Key)
		style := ""
		if svc.Instantiated {
			style = ", style=filled, fillcolor=lightblue"
		}
		_, _ = fmt.Fprintf(w, "  %q [label=%q%s];\n", svc.Key, label, style)
	}

	_, _ = fmt.Fprintln(w)

	for _, svc := range info.Services {
		for _, dep := range svc.Dependencies {
			_, _ = fmt.Fprintf(w, "  %q -> %q;\n", svc.Key, dep)
		}
	}

	_, _ = fmt.Fprintln(w, "}")
}

func (c *Container) SprintGraphDOT() string {
	var sb strings.Builder
	c.FprintGraphDOT(&sb)
	return sb.String()
}

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "*", "")
	if idx := strings.LastIndex(s, "/"); idx != -1 {
		s = s[idx+1:]
	}
	return s
}
