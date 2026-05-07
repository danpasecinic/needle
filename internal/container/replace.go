package container

import "fmt"

func (c *Container) Replace(entry *ServiceEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.registry.Remove(entry.Key)
	c.graph.RemoveNode(entry.Key)

	c.registry.Add(entry)
	c.graph.AddNode(entry.Key, entry.Dependencies)

	if len(entry.Dependencies) > 0 && c.graph.HasCycle() {
		c.registry.Remove(entry.Key)
		c.graph.RemoveNode(entry.Key)
		cyclePath := c.graph.FindCyclePath(entry.Key)
		return fmt.Errorf("circular dependency detected: %v", cyclePath)
	}

	return nil
}
