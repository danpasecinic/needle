package container

import "fmt"

func (c *Container) Replace(key string, provider ProviderFunc, dependencies []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.registry.Remove(key)
	c.graph.RemoveNode(key)

	_ = c.registry.Register(key, provider, dependencies)
	c.graph.AddNode(key, dependencies)

	if len(dependencies) > 0 && c.graph.HasCycle() {
		c.registry.Remove(key)
		c.graph.RemoveNode(key)
		cyclePath := c.graph.FindCyclePath(key)
		return fmt.Errorf("circular dependency detected: %v", cyclePath)
	}

	return nil
}

func (c *Container) ReplaceValue(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.registry.Remove(key)
	c.graph.RemoveNode(key)

	_ = c.registry.RegisterValue(key, value)
	c.graph.AddNode(key, nil)
	return nil
}
