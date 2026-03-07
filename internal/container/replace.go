package container

import "fmt"

func (c *Container) Replace(key string, provider ProviderFunc, dependencies []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.registry.Remove(key)
	c.graph.RemoveNodeUnsafe(key)

	_ = c.registry.Register(key, provider, dependencies)
	c.graph.AddNodeUnsafe(key, dependencies)

	if len(dependencies) > 0 && c.graph.HasCycleUnsafe() {
		c.registry.Remove(key)
		c.graph.RemoveNodeUnsafe(key)
		cyclePath := c.graph.FindCyclePathUnsafe(key)
		return fmt.Errorf("circular dependency detected: %v", cyclePath)
	}

	return nil
}

func (c *Container) ReplaceValue(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.registry.Remove(key)
	c.graph.RemoveNodeUnsafe(key)

	_ = c.registry.RegisterValue(key, value)
	c.graph.AddNodeUnsafe(key, nil)
	return nil
}
