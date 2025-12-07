package container

import "fmt"

func (c *Container) Replace(key string, provider ProviderFunc, dependencies []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.registry.Remove(key)
	c.graph.RemoveNode(key)

	if err := c.registry.Register(key, provider, dependencies); err != nil {
		return err
	}

	c.graph.AddNode(key, dependencies)

	if c.graph.HasCycle() {
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

	if err := c.registry.RegisterValue(key, value); err != nil {
		return err
	}

	c.graph.AddNode(key, nil)
	return nil
}
