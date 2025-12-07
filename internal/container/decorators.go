package container

import (
	"context"
	"fmt"
)

func (c *Container) AddDecorator(key string, decorator DecoratorFunc) {
	c.decoratorsMu.Lock()
	defer c.decoratorsMu.Unlock()

	c.decorators[key] = append(c.decorators[key], decorator)
}

func (c *Container) applyDecorators(ctx context.Context, key string, instance any) (any, error) {
	c.decoratorsMu.RLock()
	decorators := c.decorators[key]
	c.decoratorsMu.RUnlock()

	if len(decorators) == 0 {
		return instance, nil
	}

	var err error
	for _, decorator := range decorators {
		instance, err = decorator(ctx, c, instance)
		if err != nil {
			return nil, fmt.Errorf("decorator failed for %s: %w", key, err)
		}
	}

	return instance, nil
}
